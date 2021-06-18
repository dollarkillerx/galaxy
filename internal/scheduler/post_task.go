package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/mq_manager"
	"github.com/dollarkillerx/galaxy/internal/sync"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"

	"context"
	"log"
)

func (s *scheduler) postTask(ctx *gin.Context) {
	var task pkg.Task
	err := ctx.BindJSON(&task)
	if err != nil {
		ctx.JSON(401, pkg.ParameterError)
		return
	}
	if err := task.LegalVerification(); err != nil {
		ctx.JSON(401, pkg.StandardReturn{ErrorCode: 401, Message: err.Error()})
		return
	}

	// 初始化MQ
	err = mq_manager.Manager.Register(task)
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(401, pkg.StandardReturn{ErrorCode: 401, Message: err.Error()})
		return
	}

	cancel, cancelFunc := context.WithCancel(context.Background())
	shareSync := pkg.SharedSync{
		Task:    &task,
		Context: cancel,
		Cancel:  cancelFunc,
	}

	// core
	syncServer, err := sync.New(&shareSync)
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(401, pkg.StandardReturn{ErrorCode: 401, Message: err.Error()})
		err := mq_manager.Manager.Close(task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	err = syncServer.Monitor()
	if err != nil {
		log.Printf("%+v\n", err)
		ctx.JSON(401, pkg.StandardReturn{ErrorCode: 401, Message: err.Error()})
		err := mq_manager.Manager.Close(task.TaskID)
		if err != nil {
			log.Printf("%+v\n", err)
		}
		return
	}

	// 任务注册到数据库中
	{
		s.mu.Lock()
		defer s.mu.Unlock()

		s.taskMap[task.TaskID] = &shareSync
	}

	ctx.JSON(200, pkg.StandardReturn{Message: "success", Data: gin.H{
		"mysql_server_id": shareSync.ServerID,
		"position_name":   shareSync.PositionName,
		"position_pos":    shareSync.PositionPos,
		"task_id":         task.TaskID,
	}})
}

func (s *scheduler) getTask(ctx *gin.Context) {
	//marshal, err := json.Marshal(s.taskMap)
	//if err != nil {
	//	ctx.JSON(500, pkg.StandardReturn{ErrorCode: 500, Message: "Task acquisition failed"})
	//	return
	//}

	ctx.JSON(200, pkg.StandardReturn{Data: gin.H{
		"total": len(s.taskMap),
		"task":  s.taskMap,
	}})
}
