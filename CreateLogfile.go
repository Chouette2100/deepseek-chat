package main

import (
	"io"
	"log"
	"os"
	"time"

	//  "golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"

	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srdblib/v2"
)

// Desc: ログファイルを作成する
func CreateLogfile(version string) (file *os.File, err error) {
	// >>>>>>>>>>>>>>>>>>>>>
	// ログファイルを開く
	logfilename := version + "_" + srdblib.Version + "_" + srapi.Version + "_" +
		time.Now().Format("20060102") + ".txt"
	logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("ログファイルを開けません:", err)
		os.Exit(1)
	}
	// defer logfile.Close()

	// フォアグラウンド（端末に接続されているか）を判定
	// isForeground := terminal.IsTerminal(int(os.Stdout.Fd()))
	isForeground := term.IsTerminal(int(os.Stdout.Fd()))

	var logOutput io.Writer
	if isForeground {
		// フォアグラウンドならログファイル + コンソール
		logOutput = io.MultiWriter(os.Stdout, logfile)
	} else {
		// バックグラウンドならログファイルのみ
		logOutput = logfile
	}

	// ロガーの設定
	log.SetOutput(logOutput)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// ログ出力テスト
	log.Println("アプリケーションを起動しました")
	// <<<<<<<<<<<<<<<<<<<<

	return

}
