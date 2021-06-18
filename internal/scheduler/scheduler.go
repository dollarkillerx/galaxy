package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/config"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/gin-gonic/gin"
	"sync"

	"log"
)

// 1. 任务基础调度
// 2. 重启任务恢复

type scheduler struct {
	app     *gin.Engine
	mu      sync.Mutex
	taskMap map[string]*pkg.SharedSync // 在线task
}

func NewSchedule() *scheduler {
	// TODO： 任务恢复

	return &scheduler{
		taskMap: map[string]*pkg.SharedSync{},
	}
}

func (s *scheduler) Run() error {
	s.registerApi()

	log.Println("Galaxy run: ", config.Conf.ListenAddr)
	return s.app.Run(config.Conf.ListenAddr)
}
