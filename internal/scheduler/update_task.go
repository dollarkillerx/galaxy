package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/internal/sync_server"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"

	"context"
	"fmt"
	"log"
	"os"
	"time"
)

type StopTaskParams struct {
	TaskID   string `json:"task_id"`
	StopType string `json:"stop_type"`
}

func (s *scheduler) stopTask(ctx *gin.Context) {
	var rc StopTaskParams
	err := ctx.BindJSON(&rc)
	if err != nil {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	var task *pkg.SharedSync
	var ex bool
	s.mu.Lock()
	task, ex = s.taskMap[rc.TaskID]
	s.mu.Unlock()
	if !ex {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	if rc.StopType == "stop" {
		task.Cancel()
		task.StopSync = true // 防止重启 恢复
		task.SaveShared <- task.Task.TaskID
		ctx.JSON(200, pkg.StandardReturn{
			Message: "STOP TASK SUCCESS: " + rc.TaskID,
		})
		return
	}

	task.StopSync = false

	// 可能导致泄露
	c, cancelFunc := context.WithCancel(context.Background())

	task.Cancel = cancelFunc
	task.Context = c

	err = mq_manager.Manager.Register(*task.Task)
	if err != nil {
		ctx.JSON(500, pkg.StandardReturn{
			Message: "MQ registry error: " + rc.TaskID,
		})
		return
	}

	switch rc.StopType {
	case "recovery_v1": // v1 恢复 使用暂停时更新

	case "recovery_v2": // v2 恢复 使用最新
		task.PositionPos = 0
	}

	syncServer, err := sync_server.New(task)
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		err := mq_manager.Manager.Close(task.Task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	err = syncServer.Monitor()
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		err := mq_manager.Manager.Close(task.Task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	task.SaveShared <- task.Task.TaskID
	ctx.JSON(200, pkg.StandardReturn{
		Message: "STOP TASK SUCCESS: " + rc.TaskID,
	})
}

func (s *scheduler) deleteTask(ctx *gin.Context) {
	taskId := ctx.Param("task_id")
	if taskId == "" {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	s.mu.Lock()
	task, ex := s.taskMap[taskId]
	s.mu.Unlock()
	if !ex {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	task.Cancel()

	s.mu.Lock()
	delete(s.taskMap, taskId)
	s.mu.Unlock()

	err := storage.Storage.DelTask(taskId)
	if err != nil {
		log.Println(err)
	}

	go func() {
		// 进行回收
		time.Sleep(time.Second * 10)
		err = os.RemoveAll(fmt.Sprintf("./galaxy_schema_%s", taskId))
		if err != nil {
			log.Println(err)
		}
	}()

	ctx.JSON(200, pkg.StandardReturn{
		Message: "DEL TASK SUCCESS: " + taskId,
	})
}

func (s *scheduler) restorationTask(ctx *gin.Context) {
	param := ctx.Param("task_id")
	if param == "" {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	s.mu.Lock()
	task, ex := s.taskMap[param]
	s.mu.Unlock()
	if !ex {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	task.Cancel()
	ctx.JSON(200, pkg.StandardReturn{
		Message: "DEL TASK SUCCESS: " + param,
	})
}

func (s *scheduler) updateTask(ctx *gin.Context) {
	var update pkg.TaskUpdate
	err := ctx.BindJSON(&update)
	if err != nil {
		ctx.JSON(400, pkg.ParameterError)
		return
	}
	if err := update.LegalVerification(); err != nil {
		ctx.JSON(400, pkg.StandardReturn{ErrorCode: 400, Message: err.Error()})
		return
	}

	var task *pkg.SharedSync
	var ex bool
	{
		s.mu.Lock()
		task, ex = s.taskMap[update.TaskID]
		s.mu.Unlock()
		if !ex {
			ctx.JSON(400, pkg.ParameterError)
			return
		}

		if update.Database != "" {
			task.Task.Database = update.Database
		}
		task.Task.Tables = update.Tables
		task.Task.ExcludeTable = update.ExcludeTable

		task.Task.TablesMap = map[string]struct{}{}
		task.Task.ExcludeTableMap = map[string]struct{}{}

		for _, v := range task.Task.Tables {
			task.Task.TablesMap[v] = struct{}{}
		}

		for _, v := range task.Task.ExcludeTable {
			task.Task.ExcludeTableMap[v] = struct{}{}
		}

		task.SaveShared <- task.Task.TaskID
	}

	ctx.JSON(200, pkg.StandardReturn{Message: "Update Success"})
}
