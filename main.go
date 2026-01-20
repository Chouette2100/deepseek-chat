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

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srcom"
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
000901  Webサーバーの構成をServerConfig.ymlから読み込むようにする。
000902  デバッグライトを追加する
000903  "/"に対するハンドラをHandlerDschatからValidateJWT(HandlerDschat)に変更する。
000904  claude-sonnet-4-20250514 を追加する。
000906  textareaを操作するためのエリアを作る
000907  Caddyでの利用を考慮しhttp.ListenAndServe()も使えるようにする
000908  gemini-2.5-proと gemini-2.5-flash oのモデル名を変更する。
000909  モデル名、ベンダー名、システム名をyamlファイルから読み込むようにする。
000910  select * は使わず、カラム名を指定する。
000911  エラー発生時にデコードする前のレスポンスボディをログに出力する。
000912  000910でのmap名とカラムリストの誤りをclmlistと正す
000913  clmlistを固定化する。
000914  clmlistのidの抜けを修正する
001000  genaiパッケージを使ってgeminiに問い合わせるようにする
001001  log.Fatal()を使わず、エラーを返すようにする
*/

const version = "001001"

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

var clmlist map[string]string = map[string]string{}

func init() {
	clmlist["user"] = srdblib.ExtractStructColumns(&srdblib.User{})
	clmlist["qa_recordsDB"] = srdblib.ExtractStructColumns(&Qa_recordsDB{})
}

/*
var syslist map[string]string = map[string]string{}

func init() {
	syslist["qa_recordsDB"] = srdblib.ExtractStructColumns(&Qa_recordsDB{})
}
*/

func main() {

	//	go func() {
	//		http.ListenAndServe("localhost:6060", nil)
	//	}()

	// サーバー構成
	type ServerConfig struct {
		HTTPport string `yaml:"HTTPport"`
		SSLcrt   string `yaml:"SSLcrt"`
		SSLkey   string `yaml:"SSLkey"`
	}

	var err error

	// ログファイルを作成
	var pfile *os.File
	pfile, err = srcom.CreateLogfile3(version, srdblib.Version)
	if err != nil {
		fmt.Println("Error creating logfile")
	}
	defer pfile.Close()

	log.Printf("clmlist=%+v\n", clmlist)

	svconfig := &ServerConfig{}
	err = exsrapi.LoadConfig("ServerConfig.yml", svconfig)
	if err != nil {
		log.Printf("err=%s.\n", err.Error())
		os.Exit(1)
	}
	log.Printf("%+v\n", svconfig)

	// 設定ファイルを読み込み
	err = LoadConfig("config.yml")
	if err != nil {
		log.Printf("LoadConfig() error. err = %v\n", err)
		os.Exit(1)
	}

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
	http.HandleFunc("/", ValidateJWT(HandlerDschat))

	if svconfig.SSLcrt == "" || svconfig.SSLkey == "" {
		err = http.ListenAndServe(":"+svconfig.HTTPport, nil)
		if err != nil {
			log.Printf("ListenAndServe() error. err = %v\n", err)
		}
	} else {
		err = http.ListenAndServeTLS(":"+svconfig.HTTPport, svconfig.SSLcrt, svconfig.SSLkey, nil)
		log.Printf("SSL enabled. Listening on port %s\n", svconfig.HTTPport)
		if err != nil {
			log.Printf("ListenAndServe() error. err = %v\n", err)
		}
	}

}
