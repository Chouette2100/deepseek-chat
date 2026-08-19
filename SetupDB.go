package main

import (
	"log"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/Chouette2100/srdblib/v3"
	"github.com/go-gorp/gorp"
)

type User struct {
	Email string
	Pswd  string
	Ts    time.Time
}

var db *sql.DB
var dbmap *gorp.DbMap

// Desc: データベース接続を設定する
func SetupDB() (err error) {
	var dbconfig *srdblib.DBConfig
	// >>>>>>>>>>>>>>>>>>>>>
	// データベース接続
	db, dbconfig, err = srdblib.OpenDb("DBConfig.enc.yml")
	if err != nil {
		log.Printf("Database error. err = %v\n", err)
		return
	}
	if dbconfig.UseSSH {
		defer srdblib.Dialer.Close()
	}
	// defer srdblib.Db.Close() // ここで閉じると他のパッケージで使えなくなる

	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	dbmap = &gorp.DbMap{Db: db,
		Dialect:         dial,
		ExpandSliceArgs: true, //スライス引数展開オプションを有効化する
	}
	dbmap.AddTableWithName(Qa_recordsDB{}, "qa_records").SetKeys(true, "Id")
	dbmap.AddTableWithName(User{}, "user").SetKeys(true, "Email")
	// <<<<<<<<<<<<<<<<<<<<

	return
}
