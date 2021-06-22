# Galaxy
Galaxy High performance MySQL CDC

![](./doc/l1.png)

## Features
- [x] Infrastructure
- [x] Kafka
- [ ] NSQ
- [ ] Monitorable   
- [x] Recovery synchronization
- [x] Pause synchronization
- [x] Latest synchronization
- [ ] Compliance Test


## depend
- github.com/go-mysql-org/go-mysql

## use
###  mysql configure
Server Config: Ensure server_id is set, and that row-based replication is on.
``` 
$ vi /etc/mysql/my.cnf

[mysqld]
server_id=1
log-bin=master
binlog_format=row
```
Or on a running server:
``` 
mysql> set global binlog_format=ROW;
mysql> set global binlog_row_image=FULL;
```
note: binlog_format is a session-based property. You will need to shutdown all active connections to fully convert to row-based replication.

Permissions: Galaxy needs permissions to act as a replica, and to write to the galaxy database.
``` 
mysql> CREATE USER 'galaxy'@'%' IDENTIFIED BY 'you_password';
mysql> GRANT ALL ON galaxy.* TO 'galaxy'@'%';
mysql> GRANT SELECT, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'galaxy'@'%';

# or for running galaxy locally:

mysql> CREATE USER 'galaxy'@'localhost' IDENTIFIED BY 'you_password';
mysql> GRANT ALL ON galaxy.* TO 'galaxy'@'localhost';
mysql> GRANT SELECT, REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'galaxy'@'localhost';
```

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


