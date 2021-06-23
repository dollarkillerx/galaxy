package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/config"
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/internal/sync_server"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"

	"context"
	"log"
	"sync"
)

// 1. 任务基础调度
// 2. 重启任务恢复

// TODO: [x] 状态持久化
// TODO: [x] 任务恢复
// TODO: [x] 任务多暂停模式

type scheduler struct {
	app     *gin.Engine
	mu      sync.Mutex
	taskMap map[string]*pkg.SharedSync // 在线task

	saveChan chan string
}

func NewSchedule() *scheduler {

	return &scheduler{
		taskMap:  map[string]*pkg.SharedSync{},
		saveChan: make(chan string, 100),
	}
}

func (s *scheduler) Run() error {
	s.registerApi()
	go s.durability()
	// TODO： 任务恢复
	s.taskRecovery()

	log.Println("Galaxy run: ", config.Conf.ListenAddr)
	return s.app.Run(config.Conf.ListenAddr)
}

// 持久化
func (s *scheduler) durability() {
loop:
	for {
		select {
		case taskID, ex := <-s.saveChan:
			if !ex {
				break loop
			}

			sharedSync, ex := s.taskMap[taskID]
			if ex {
				{
					sharedSync.Rw.Lock()
					err := storage.Storage.UpdateTask(taskID, *sharedSync) // 可能存在 内存逃逸
					sharedSync.Rw.Unlock()
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func (s *scheduler) taskRecovery() {
	tasks, err := storage.Storage.GetTasks()
	if err != nil {
		log.Println(err)
		return
	}

	if len(tasks) != 0 {
		log.Println("Start the task recovery, the number of tasks: ", len(tasks))

		for i := range tasks {
			it := tasks[i]
			s.taskMap[it.Task.TaskID] = it

			// 暂停任务 不自动启动
			if it.StopSync {
				continue
			}

			cancel, cancelFunc := context.WithCancel(context.Background())
			it.Cancel = cancelFunc
			it.Context = cancel
			it.SaveShared = s.saveChan

			err := mq_manager.Manager.Register(*it.Task)
			if err != nil {
				it.ErrorMsg = err.Error()
				log.Println("Manager Error Task recovery failed: ", it.Task.TaskID, "  err: ", err)
				return
			}

			s, err := sync_server.New(it)
			if err != nil {
				log.Printf("Task recovery failed:%s ,%+v\n", it.Task.TaskID, err)
				it.ErrorMsg = err.Error()
				return
			}
			err = s.Monitor()
			if err != nil {
				log.Printf("Task recovery failed:%s ,%+v\n", it.Task.TaskID, err)
				it.ErrorMsg = err.Error()
				return
			}

			log.Println("Task recovery success: ", it.Task.TaskID)
		}
	}
}
