package main

import (
	// "encoding/json"
	"fmt"
	"log"
	"net/http"
	// _ "net/http/pprof"
	"os"
	// "strconv"
	"time"

	// "github.com/gin-gonic/gin"
	// "github.com/joho/godotenv"

	"github.com/Chouette2100/srdblib/v2"
)

/*
000300  前提と必要な履歴の送信機能を作成する
000400  deepseek, claude, gemini, openaiのAPIを使う機能を作成する
000500  "o3-mini-2025-01-31"ではmax_tokensが使えないので、max_completion_tokensに変更
000600  JWT認証を追加する（Github Copilot(GPT-4o)による
000700  モデルとして gemini-2.5-pro-preview-05-06 及び gemini-2.5-flash-preview-04-17 を追加する。
000800  gemini の場合、レスポンスのpartsの配列のすべてを取得する。
000801  HandlerDschat()でのQa.MaxtokensとQa.Modelnameの初期値を20000とgemini-2.5-flash-preview-04-17に変更する。
*/

const version = "000801"

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

// データベースの構造体　※ 修正注意
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
	// vederlistのApikeyを環境変数から取得
	for k := range venderlist {
		v := venderlist[k]
		v.Apikey = os.Getenv(v.EvAPI)
		venderlist[k] = v
	}
}

func main() {

	//	go func() {
	//		http.ListenAndServe("localhost:6060", nil)
	//	}()

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

	http.HandleFunc("/dschat", ValidateJWT(HandlerDschat))
	http.HandleFunc("/signup", SignupHandler)
	http.HandleFunc("/verify", VerifyCodeHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/", HandlerDschat)

	sport := os.Getenv("SPORT")
	if sport == "" {
		sport = "8082"
	}

	// err = http.ListenAndServe(":"+sport, nil)
	err = http.ListenAndServeTLS(":"+sport, "/home/chouette/.ssh/cert.pem", "/home/chouette/.ssh/key.pem", nil)
	if err != nil {
		log.Printf("ListenAndServe() error. err = %v\n", err)
	}

}
