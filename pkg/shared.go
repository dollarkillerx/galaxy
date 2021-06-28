package pkg

import (
	"context"
	"sync"
)

// SharedSync 数据共享结构 服务状态同步
type SharedSync struct {
	Rw           sync.RWMutex       `json:"-"`         // 使用场景 读多写少 多任务进行变更
	Task         *Task              `json:"task"`      // 任务更新同步
	ServerID     uint32             `json:"server_id"` // mysql server_id
	PositionName string             `json:"position_name"`
	PositionPos  uint32             `json:"position_pos"`
	Context      context.Context    `json:"-"` // 结束任务
	Cancel       context.CancelFunc `json:"-"`
	StopSync     bool               `json:"stop_sync"` // 暂停任务
	ErrorMsg     string             `json:"error_msg"`
	SaveShared   chan string        `json:"-"` // 更新存储

	ConcurrentlyTask []*ConcurrentlyTask `json:"concurrently_task_manager"` // 构成  old, new (注意 断电 消息可能重发)
}

type ConcurrentlyTask struct {
	PosName string `json:"pos_name"` // pos file name
	Pos     uint32 `json:"pos"`      // 当前任务pos
	Success bool   `json:"success"`  // 当前任务是否成功
}
