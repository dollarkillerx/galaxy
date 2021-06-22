package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
	"github.com/dollarkillerx/galaxy/internal/sync_server"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"

	"context"
	"fmt"
	"log"
)

func (s *scheduler) postTask(ctx *gin.Context) {
	var task pkg.Task
	err := ctx.BindJSON(&task)
	if err != nil {
		ctx.JSON(400, pkg.ParameterError)
		return
	}
	if err := task.LegalVerification(); err != nil {
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		return
	}

	// 初始化MQ
	err = mq_manager.Manager.Register(task)
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		return
	}

	cancel, cancelFunc := context.WithCancel(context.Background())
	shareSync := pkg.SharedSync{
		Task:       &task,
		Context:    cancel,
		Cancel:     cancelFunc,
		SaveShared: s.saveChan,
	}

	// core
	syncServer, err := sync_server.New(&shareSync)
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		err := mq_manager.Manager.Close(task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	err = syncServer.Monitor()
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		err := mq_manager.Manager.Close(task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	// 任务注册到数据库中
	s.mu.Lock()
	_, ex := s.taskMap[task.TaskID]
	s.mu.Unlock()
	if ex {
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: fmt.Sprintf("Taskid is existed for %s tasks", task.TaskID)})
	}

	s.mu.Lock()
	s.taskMap[task.TaskID] = &shareSync
	s.mu.Unlock()

	shareSync.SaveShared <- task.TaskID // 进行持久化存储

	ctx.JSON(200, pkg.StandardReturn{Message: "success", Data: gin.H{
		"mysql_server_id": shareSync.ServerID,
		"position_name":   shareSync.PositionName,
		"position_pos":    shareSync.PositionPos,
		"task_id":         task.TaskID,
	}})
}

func (s *scheduler) getTask(ctx *gin.Context) {
	ctx.JSON(200, pkg.StandardReturn{Data: gin.H{
		"total": len(s.taskMap),
		"task":  s.taskMap,
	}})
}
