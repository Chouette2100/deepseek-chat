package main

import (
	// "encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	// "strconv"
	"time"

	// "github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Chouette2100/srdblib/v2"
)

/*
000300  前提と必要な履歴の送信機能を作成する
*/

const version = "000300"

type CustomTime time.Time

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(ct).Format("2006-01-02 15:04:05") + `"`), nil
}

func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	// ダブルクォートを取り除く
	str := string(data[1 : len(data)-1])
	// カスタムフォーマットで時刻をパース
	t, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		return err
	}
	*ct = CustomTime(t)
	return nil
}

// データベースの構造体
type Qa_recordsDB struct {
        Id           int       `json:"id"`
        Timestamp    time.Time `json:"timestamp"`
        Responsetime int64     `json:"responsetime"`
        Modelname    string    `json:"model_name"`
        Maxtokens    int       `json:"maxtokens"`
        Temperature  float64   `json:"temperature"`
        System       string    `json:"system"`
        Question     string    `json:"question"`
        Answer       string    `json:"answer"`
        Itokens      int       `json:"itokens"`
        Otokens      int       `json:"otokens"`
        StopReason   string    `json:"stop_reason"`
}


func init() {
	// .envファイルを読み込む
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		log.Fatal(err)
	}
}

func main() {

	var err error

	// ログファイルを作成
	var pfile *os.File
	pfile, err = CreateLogfile(version)
	if err != nil {
		fmt.Println("Error creating logfile")
	}
	defer pfile.Close()

	// データベース接続
	err = SetupDB()
	if err != nil {
		log.Printf("SetupDB() error. err = %v\n", err)
	}
	defer srdblib.Db.Close()

	http.HandleFunc("/dschat", HandlerDschat)

	err = http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Printf("ListenAndServe() error. err = %v\n", err)
	}

}
