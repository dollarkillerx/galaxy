package concurrently_manager

import (
	"encoding/json"
	"log"

	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/dollarkillerx/go-mysql/mysql"
)

// ConcurrentlyTaskManager 并发任务管理器
type ConcurrentlyTaskManager struct {
	sharedSync *pkg.SharedSync

	pos     uint32
	recover bool // 当前系统是否处于 恢复状态
}

func InitConcurrentlyTaskManager(sharedSync *pkg.SharedSync) *ConcurrentlyTaskManager {
	tm := &ConcurrentlyTaskManager{
		//Tasks:
		sharedSync: sharedSync,
	}

	// 兼容老版本
	if sharedSync.ConcurrentlyTask == nil {
		sharedSync.ConcurrentlyTask = make([]*pkg.ConcurrentlyTask, 0)
	}
	return tm
}

// GetPos 获取pos
func (c *ConcurrentlyTaskManager) GetPos() (mysql.Position, bool) {
	if c.sharedSync.PositionPos == 0 { // 使用最新的
		return mysql.Position{}, false
	} else if len(c.sharedSync.ConcurrentlyTask) != 0 {
		c.recover = true
		c.pos = c.sharedSync.ConcurrentlyTask[len(c.sharedSync.ConcurrentlyTask)-1].Pos
		c.sharedSync.ConcurrentlyTaskBack = c.sharedSync.ConcurrentlyTask
		marshal, err := json.Marshal(c.sharedSync.ConcurrentlyTask)
		if err == nil {
			log.Println(string(marshal))
			//return mysql.Position{}, false
		}
		pos := c.sharedSync.ConcurrentlyTask[0].Pos
		posName := c.sharedSync.ConcurrentlyTask[0].PosName
		log.Println("Pos ConcurrentlyTask Recovery", posName, pos, "   taskID: ", c.sharedSync.Task.TaskID)
		return mysql.Position{
			Name: posName,
			Pos:  pos,
		}, true
	} else if c.sharedSync.PositionPos != 0 {
		log.Println("Pos Recovery", c.sharedSync.PositionName, c.sharedSync.PositionPos, "   taskID: ", c.sharedSync.Task.TaskID)
		// 使用设定值
		return mysql.Position{
			Name: c.sharedSync.PositionName,
			Pos:  c.sharedSync.PositionPos,
		}, true
	}

	log.Println("ConcurrentlyTaskManager GetPos ?????????")
	return mysql.Position{}, false
}

// SendTask 线程下发任务
//func (c *ConcurrentlyTaskManager) SendTask(fn async_utils.PoolFunc) {
//	c.poolFund.Send(fn)
//}

// RecordStartState 记录任务开始状态
func (c *ConcurrentlyTaskManager) RecordStartState(posName string, pos uint32) {
	if len(c.sharedSync.ConcurrentlyTask) == 2 { //  ConcurrentlyTask len max 2
		c.sharedSync.ConcurrentlyTask = c.sharedSync.ConcurrentlyTask[1:]
	}

	c.sharedSync.ConcurrentlyTask = append(c.sharedSync.ConcurrentlyTask,
		&pkg.ConcurrentlyTask{PosName: posName, Pos: pos})

	c.sharedSync.SaveShared <- c.sharedSync.Task.TaskID
}

// MissionComplete 记录任务完毕状态
func (c *ConcurrentlyTaskManager) MissionComplete(posName string, pos uint32) {
	//c.sharedSync.ConcurrentlyTask = append(c.sharedSync.ConcurrentlyTask,
	//	&pkg.ConcurrentlyTask{PosName: posName, Pos: pos})

	for i := range c.sharedSync.ConcurrentlyTask {
		r := c.sharedSync.ConcurrentlyTask[i]
		if r.PosName == posName && r.Pos == pos {
			r.Success = true
		}
	}
	c.sharedSync.SaveShared <- c.sharedSync.Task.TaskID
}

// Continue 任务恢复时 跳过已处理的log
func (c *ConcurrentlyTaskManager) Continue(offset uint32) bool {
	if !c.recover { // 检测是否处于 恢复状态中
		return false
	}

	// 恢复
	for _, v := range c.sharedSync.ConcurrentlyTaskBack {
		// 判定是否跳过
		if v.Pos == offset && v.Success == false {
			//log.Println("跳过 ...............")
			//fmt.Println("Is True: ", v.Pos)
			return false
		}
	}

	// 当任务完成时 解除恢复状态
	if offset > c.pos {
		//fmt.Println("Is Over: ", offset, "  pos: ", c.pos)
		c.recover = false
		return false
	}

	return true
}

//// gc 处理已完成任务
//func (c *ConcurrentlyTaskManager) gcManager() {
//	for {
//		select {
//		case <-time.NewTicker(time.Second * 10).C:
//			c.gc()
//		}
//	}
//}

//// 处理遗留任务
//func (c *ConcurrentlyTaskManager) gc() {
//	log.Println("ConcurrentlyTaskManager GC Task: ", c.sharedSync.Task.TaskID)
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	// 排序
//	sort.Sort(c)
//
//	// 构造新的tasks
//	var tasks []*pkg.ConcurrentlyTask
//	for i := range c.sharedSync.ConcurrentlyTask {
//		if !c.sharedSync.ConcurrentlyTask[i].Success {
//			tasks = append(tasks, c.sharedSync.ConcurrentlyTask[i])
//		}
//	}
//
//	oldLen := len(c.sharedSync.ConcurrentlyTask)
//	newLen := len(tasks)
//	// 当没有改变时 说明任务都在执行 无需要清洗
//	if oldLen == newLen {
//		return
//	}
//
//	// 反之更改数据 并持久化
//	c.sharedSync.ConcurrentlyTask = tasks
//
//	c.sharedSync.SaveShared <- c.sharedSync.Task.TaskID
//}
//
//func (c *ConcurrentlyTaskManager) Len() int {
//	return len(c.sharedSync.ConcurrentlyTask)
//}
//
//func (c *ConcurrentlyTaskManager) Swap(i, j int) {
//	c.sharedSync.ConcurrentlyTask[i], c.sharedSync.ConcurrentlyTask[j] = c.sharedSync.ConcurrentlyTask[j], c.sharedSync.ConcurrentlyTask[i]
//}
//
//func (c *ConcurrentlyTaskManager) Less(i, j int) bool {
//	return c.sharedSync.ConcurrentlyTask[i].Pos < c.sharedSync.ConcurrentlyTask[j].Pos
//}
//
//func (c *ConcurrentlyTaskManager) Marshal() ([]byte, error) {
//	return json.Marshal(c)
//}
