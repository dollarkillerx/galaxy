package pkg

// BaseConfig

type KafkaConf struct {
	EnableSASL bool     `json:"enable_sasl"`
	Brokers    []string `json:"brokers"`
	User       string   `json:"user"`
	Password   string   `json:"password"`
	Topic      string   `json:"topic"`
}

type NsqConf struct {
}

type MongoDBConf struct {
}

type ESConf struct {
}

// MQEvent

type MQEvent struct {
	Database    string                 `json:"database"`
	Table       string                 `json:"table"`
	Action      string                 `json:"action"`
	Before      map[string]interface{} `json:"before"`
	After       map[string]interface{} `json:"after"`
	OrgRow      [][]interface{}        `json:"org_row"`
	EventHeader EventHeader            `json:"event_header"`
}

type EventHeader struct {
	Timestamp uint32 `json:"timestamp"`
	LogPos    uint32 `json:"log_pos"`
}
