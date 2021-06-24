package test

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/dollarkillerx/async_utils"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

type TestModule struct {
	gorm.Model
	Username string   `json:"username"`
	Age      int      `json:"age"`
	Money    float64  `gorm:"type:DECIMAL(20,4)"`
	Total    *float64 `gorm:"type:FLOAT(20,4)"`
	//Sr       string   `json:"sr"`
	//Ssr      string   `json:"ssr"`
}

var db *gorm.DB
var err error

func init() {
	dsn := "root:root@tcp(192.168.88.11:3307)/test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
}

func TestBet(t *testing.T) {
	err := db.AutoMigrate(&TestModule{})
	if err != nil {
		log.Fatalln(err)
	}

	over := make(chan struct{})
	poolFunc := async_utils.NewPoolFunc(10, func() {
		close(over)
	})

	for j := 0; j < 100000; j++ {
		var r []TestModule
		for i := 0; i < 1000; i++ {
			f := rand.Float64()
			r = append(r, TestModule{
				Username: fmt.Sprintf("sp7_v%d_%d", j, i),
				Age:      i,
				Money:    f,
				Total:    &f,
				//Sp:       fmt.Sprintf("sp_v%d_%d", j, i),
				//Ssr:      fmt.Sprintf("ssr_v%d_%d", j, i),
			})
		}

		poolFunc.Send(func() {
			db.Create(r)
			fmt.Println("In: ", j)
		})
	}

	poolFunc.Over()
	<-over

	//err = db.CreateInBatches(&r, 600).Error
	//if err != nil {
	//	log.Fatalln(err)
	//}
}

func TestUpdate(t *testing.T) {
	var lis []TestModule
	err := db.Model(&TestModule{}).Limit(200).Scan(&lis).Error
	if err != nil {
		log.Fatalln(err)
	}

	for i, v := range lis {
		rand.Seed(time.Now().UnixNano())
		intn := rand.Intn(100)
		if intn > 50 {
			err := db.Model(&TestModule{}).Where("id = ?", v.ID).Unscoped().Delete(&TestModule{}).Error
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			err := db.Model(&TestModule{}).Where("id = ?", v.ID).Update("username", fmt.Sprintf("%d_sz", i)).Error
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}
