package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/storage"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"

	"log"
)

func (s *scheduler) stopTask(ctx *gin.Context) {
	param := ctx.Param("task_id")
	var task *pkg.SharedSync
	var ex bool
	s.mu.Lock()
	task, ex = s.taskMap[param]
	s.mu.Unlock()
	if !ex {
		ctx.JSON(400, pkg.ParameterError)
		return
	}

	task.Rw.Lock()
	task.StopSync = true
	task.Rw.Unlock()

	ctx.JSON(200, pkg.StandardReturn{
		Message: "STOP TASK SUCCESS: " + param,
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

	err := storage.Storage.DelTask(taskId)
	if err != nil {
		log.Println(err)
	}

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

		task.Task.Database = update.Database
		task.Task.Tables = update.Tables
		task.Task.ExcludeTable = update.ExcludeTable
	}

	ctx.JSON(200, pkg.StandardReturn{Message: "Update Success"})
}
