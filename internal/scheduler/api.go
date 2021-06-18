package scheduler

import "github.com/gin-gonic/gin"

func (s *scheduler) registerApi() error {
	app := gin.New()

	v1 := app.Group("/v1")
	{
		// 1. 下发任务
		// 2. 获取任务
		// 3. 暂停任务
		// 4. 修改任务
		// 5. 删除任务
		v1.POST("/")

	}

	return nil
}
