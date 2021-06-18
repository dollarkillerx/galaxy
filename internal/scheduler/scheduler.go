package scheduler

import "github.com/gin-gonic/gin"

// 1. 任务基础调度
// 2. 重启任务恢复

type scheduler struct {
	app *gin.Engine
}

func NewSchedule() *scheduler {
	return &scheduler{}
}

func (s *scheduler) Run() error {
	return nil
}
