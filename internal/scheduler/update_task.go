package scheduler

import (
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"
)

func (s *scheduler) stopTask(ctx *gin.Context) {
	param := ctx.Param("task_id")
	var task *pkg.SharedSync
	var ex bool
	{
		s.mu.Lock()
		defer s.mu.Unlock()
		task, ex = s.taskMap[param]
		if !ex {
			ctx.JSON(401, pkg.ParameterError)
			return
		}
	}

	{
		task.Rw.Lock()
		defer task.Rw.Unlock()

		task.StopSync = true
	}

	ctx.JSON(200, pkg.StandardReturn{
		Message: "STOP TASK SUCCESS: " + param,
	})
}

func (s *scheduler) deleteTask(ctx *gin.Context) {
	param := ctx.Param("task_id")
	if param == "" {
		ctx.JSON(401, pkg.ParameterError)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	task, ex := s.taskMap[param]
	if !ex {
		ctx.JSON(401, pkg.ParameterError)
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
		ctx.JSON(401, pkg.ParameterError)
		return
	}
	if err := update.LegalVerification(); err != nil {
		ctx.JSON(401, pkg.StandardReturn{ErrorCode: 401, Message: err.Error()})
		return
	}

	var task *pkg.SharedSync
	var ex bool
	{
		s.mu.Lock()
		defer s.mu.Unlock()
		task, ex = s.taskMap[update.TaskID]
		if !ex {
			ctx.JSON(401, pkg.ParameterError)
			return
		}

		task.Task.Database = update.Database
		task.Task.Tables = update.Tables
		task.Task.ExcludeTable = update.ExcludeTable
	}

	ctx.JSON(200, pkg.StandardReturn{Message: "Update Success"})
}
