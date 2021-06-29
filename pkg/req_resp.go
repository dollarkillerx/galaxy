package pkg

import "github.com/pingcap/errors"

type StandardReturn struct {
	ErrorCode int         `json:"error_code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

type Task struct {
	TaskBaseData
	KafkaConf   *KafkaConf   `json:"kafka_conf"`
	NsqConf     *NsqConf     `json:"nsq_conf"`
	MongoDBConf *MongoDBConf `json:"mongo_db_conf"`
	ESConf      *ESConf      `json:"es_conf"`
}

func (t *Task) LegalVerification() error {
	if t.TaskID == "" {
		return errors.New("task is null")
	}
	//if len(t.Database) == 0 {
	//	return errors.New("database is null")
	//}

	t.DatabaseMap = make(map[string]struct{})
	t.TablesMap = make(map[string]struct{})
	t.ExcludeTableMap = make(map[string]struct{})

	for _, v := range t.Database {
		t.DatabaseMap[v] = struct{}{}
	}

	for _, v := range t.Tables {
		t.TablesMap[v] = struct{}{}
	}

	for _, v := range t.ExcludeTable {
		t.ExcludeTableMap[v] = struct{}{}
	}
	return nil
}

type TaskUpdate struct {
	TaskID       string   `json:"task_id"`
	Database     []string `json:"database"`
	Tables       []string `json:"tables"`        // default: all table
	ExcludeTable []string `json:"exclude_table"` // 排除表 table
}

// TODO: 添加校验
func (t *TaskUpdate) LegalVerification() error {
	if t.TaskID == "" {
		return errors.New("task is null")
	}
	//if len(t.Database) == 0 {
	//	return errors.New("database is null")
	//}
	return nil
}

type TaskBaseData struct {
	TaskID          string              `json:"task_id"`
	MySqlConfig     MySQLConfig         `json:"mysql_config"`
	Database        []string            `json:"database"`
	Tables          []string            `json:"tables"`        // default: all table
	ExcludeTable    []string            `json:"exclude_table"` // 排除表 table
	DatabaseMap     map[string]struct{} `json:"database_map"`
	TablesMap       map[string]struct{} `json:"tables_map"`
	ExcludeTableMap map[string]struct{} `json:"exclude_table_map"`
	//StartTime       uint32              `json:"start_time"` // default: Use the latest (鸡肋非必要不建议使用, 使用限制: 1. start_time时间到生成任务 时段schema 不可发生改变, 2. 性能低效)
}

type MySQLConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
}
