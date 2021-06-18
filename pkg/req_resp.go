package pkg

type Task struct {
	TaskBaseData
	KafkaConf   *KafkaConf   `json:"kafka_conf"`
	NsqConf     *NsqConf     `json:"nsq_conf"`
	MongoDBConf *MongoDBConf `json:"mongo_db_conf"`
	ESConf      *ESConf      `json:"es_conf"`
}

type TaskBaseData struct {
	TaskID      string      `json:"task_id"`
	MySqlConfig MySQLConfig `json:"mysql_config"`
	Database    string      `json:"database"`
	Tables      []string    `json:"tables"`       // default: all table
	ShieldTable []string    `json:"shield_table"` // 禁用table
	StartTime   uint32      `json:"start_time"`   // default: Use the latest
}

type MySQLConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
}
