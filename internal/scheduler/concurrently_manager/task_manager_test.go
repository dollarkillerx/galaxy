package concurrently_manager

import (
	"fmt"
	"sort"
	"testing"
)

func TestTaskManager(t *testing.T) {
	manager := InitConcurrentlyTaskManager()
	manager.Tasks = append(manager.Tasks,
		&ConcurrentlyTask{Pos: 6},
		&ConcurrentlyTask{Pos: 1},
		&ConcurrentlyTask{Pos: 3},
		&ConcurrentlyTask{Pos: 5},
		&ConcurrentlyTask{Pos: 90},
	)

	marshal, err := manager.Marshal()
	if err == nil {
		fmt.Println(string(marshal))
	}

	sort.Sort(manager)

	marshal, err = manager.Marshal()
	if err == nil {
		fmt.Println(string(marshal))
	}
}
