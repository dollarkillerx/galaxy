package scheduler

import (
	"github.com/dollarkillerx/galaxy/internal/prometheus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *scheduler) registerApi() {
	s.app = gin.New()

	v1 := s.app.Group("/v1")
	{
		// 1. 下发任务
		v1.POST("/post_task", s.postTask)
		// 2. 获取任务
		v1.GET("/task", s.getTask)
		// 3. 暂停任务
		v1.POST("/stop_task/:task_id", s.stopTask) // stop_task 不会恢复到之前的log位置 而是获取当前最新的log
		// 4. 修改任务
		v1.POST("/update_task", s.updateTask)
		// 5. 删除任务
		v1.POST("/delete_task/:task_id", s.deleteTask)
		// 5. 任务修复  (尝试修复任务)
		v1.POST("/restoration_task/:task_id", s.restorationTask)
	}

	prometheus.Init()
	// prometheus
	s.app.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
