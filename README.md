# Galaxy
Galaxy High performance MySQL CDC

![](./doc/l1.png)

### Features
- [ ] Infrastructure
- [ ] Kafka
- [ ] NSQ
- [ ] Monitorable   
- [ ] Recovery synchronization
- [ ] Pause synchronization
- [ ] Latest synchronization


### depend
- github.com/go-mysql-org/go-mysql

### use
- new cdc task:
``` 
type Task struct {
	TaskID      string   `json:"task_id"`
	MySQLUri    string   `json:"mysql_uri"` // user:password@ip:port
	Database    string   `json:"database"`
	Tables      []string `json:"tables"` // default: all table
	ShieldTable []string `json:"shield_table"`
	StartTime   int64    `json:"start_time"` // default: Starting from 0  , Use the latest: 1

	KafkaConf   *KafkaConf   `json:"kafka_conf"`
	NsqConf     *NsqConf     `json:"nsq_conf"`
	MongoDBConf *MongoDBConf `json:"mongo_db_conf"`
	ESConf      *ESConf      `json:"es_conf"`
}
```