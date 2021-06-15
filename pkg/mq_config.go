package pkg

// BaseConfig

type KafkaConf struct {
}

type NsqConf struct {
}

type MongoDBConf struct {
}

type ESConf struct {
}

// MQEvent

type MQEvent struct {
	Table       *Task           `json:"table"`
	Action      string          `json:"action"`
	JSONRow     string          `json:"json_row"`
	OrgRow      [][]interface{} `json:"org_row"`
	EventHeader string          `json:"event_header"`
}

type Table struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Rows     string `json:"rows"`
}

type EventHeader struct {
	Timestamp uint32 `json:"timestamp"`
	LogPos    uint32 `json:"log_pos"`
}
