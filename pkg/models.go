package pkg

type Task struct {
	TaskID      string   `json:"task_id"`
	MySQLUri    string   `json:"my_sql_uri"` // user:password@ip:port
	Database    string   `json:"database"`
	Tables      []string `json:"tables"` // default: all table
	ShieldTable []string `json:"shield_table"`
	StartTime   int64    `json:"start_time"` // default: Starting from 0  , Use the latest: 1

	KafkaConf   *KafkaConf   `json:"kafka_conf"`
	NsqConf     *NsqConf     `json:"nsq_conf"`
	MongoDBConf *MongoDBConf `json:"mongo_db_conf"`
	ESConf      *ESConf      `json:"es_conf"`
}
