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

// TODO: 添加校验
func (t *Task) LegalVerification() error {
	if t.TaskID == "" {
		return errors.New("task is null")
	}
	if t.Database == "" {
		return errors.New("database is null")
	}
	return nil
}

type TaskUpdate struct {
	TaskID       string   `json:"task_id"`
	Database     string   `json:"database"`
	Tables       []string `json:"tables"`       // default: all table
	ExcludeTable []string `json:"shield_table"` // 排除表 table
}

// TODO: 添加校验
func (t *TaskUpdate) LegalVerification() error {
	if t.TaskID == "" {
		return errors.New("task is null")
	}
	if t.Database == "" {
		return errors.New("database is null")
	}
	return nil
}

type TaskBaseData struct {
	TaskID       string      `json:"task_id"`
	MySqlConfig  MySQLConfig `json:"mysql_config"`
	Database     string      `json:"database"`
	Tables       []string    `json:"tables"`       // default: all table
	ExcludeTable []string    `json:"shield_table"` // 排除表 table
	StartTime    uint32      `json:"start_time"`   // default: Use the latest
}

type MySQLConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
}
