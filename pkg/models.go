package pkg

type MySQLStatus struct {
	File              string
	Position          uint32
	Binlog_Do_DB      string
	Binlog_lgnore_DB  string
	Executed_Gtid_Set string
}

type MySQLSchema struct {
	Field      string
	Type       string
	Collation  *string
	Null       string
	Key        *string
	Default    *string
	Extra      *string
	Privileges string
	Comment    *string
}

// HistorySchemas mem <=> db 需要内存和本地存储互相映射
// 不用太过于强调binlog schema, history deltas, new deltas
type HistorySchemas struct {
	Db    string `json:"db"`
	Table string `json:"table"`

	Deltas Deltas `json:"deltas"`
}

type Deltas struct {
	Old DeltasItem `json:"old"`
	Def DeltasItem `json:"def"`
}

type DeltasItem struct {
	Database string    `json:"database"`
	Table    string    `json:"table"`
	Columns  []Columns `json:"columns"`
}

type Columns struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	NotNull bool   `json:"not_null"` // 非空
}
