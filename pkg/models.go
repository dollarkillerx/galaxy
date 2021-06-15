package pkg

type MySQLStatus struct {
	File              string
	Position          uint32
	Binlog_Do_DB      string
	Binlog_lgnore_DB  string
	Executed_Gtid_Set string
}
