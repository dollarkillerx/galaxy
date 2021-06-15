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
	Collation  string
	Null       string
	Key        string
	Default    *string
	Extra      string
	Privileges string
	Comment    string
}
