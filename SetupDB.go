package main

import (
	"github.com/Chouette2100/srdblib/v2"
	"github.com/go-gorp/gorp"
	"log"
)

// Desc: データベース接続を設定する
func SetupDB() (err error) {
	// >>>>>>>>>>>>>>>>>>>>>
	// データベース接続
	dbconfig, err := srdblib.OpenDb("DBConfig.yml")
	if err != nil {
		log.Printf("Database error. err = %v\n", err)
		return
	}
	if dbconfig.UseSSH {
		defer srdblib.Dialer.Close()
	}
	// defer srdblib.Db.Close() // ここで閉じると他のパッケージで使えなくなる

	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	srdblib.Dbmap = &gorp.DbMap{Db: srdblib.Db,
		Dialect:         dial,
		ExpandSliceArgs: true, //スライス引数展開オプションを有効化する
	}
	srdblib.Dbmap.AddTableWithName(Qa_recordsDB{}, "qa_records").SetKeys(true, "Id")
	// <<<<<<<<<<<<<<<<<<<<

	return
}
