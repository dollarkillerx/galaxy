package test

import (
	"database/sql"
	"fmt"
	"log"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestBatchInsert(t *testing.T) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@(%s:%d)/mysql", "root", "root", "192.168.88.11", 3307))
	if err != nil {
		log.Fatalln(err)
	}

	begin, err := db.Begin()
	if err != nil {
		log.Fatalln(err)
	}

	for i := 300; i < 400; i++ {
		_, err := begin.Exec(fmt.Sprintf("INSERT into `casbin_rule`  (p_type,v0,v2) values (\"p_type_%d\",\"v0_%d\",\"v2_%d\"), (\"p_type1_%d\",\"v01_%d\",\"v21_%d\") ,(\"p_type2_%d\",\"v02_%d\",\"v22_%d\")", i, i, i, i, i, i, i, i, i))
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = begin.Commit()
	if err != nil {
		log.Fatalln()
	}
}

func TestP2(t *testing.T) {
	c := []int{1, 2, 3, 4, 5, 6, 7, 8}
	for i := 0; i < len(c); i += 2 {
		fmt.Println(c[i], " ", c[i+1])
	}
}
