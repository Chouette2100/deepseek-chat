// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	// "os"
	"strconv"
	"strings"
	"time"

	"github.com/Chouette2100/srdblib/v2"
)

type qah struct {
	Sid      string
	Model    string
	Question string
	Answer   string
}

var history []qah

type Modeltype struct {
	Model  string
	Vendor string
}

var systemlist map[string]string = map[string]string{
	"none":    "",
	"Go":      "あなたはGoのエクスパート。環境はLinuxMint21.3、Go1.23.1、net/http＋go-template、DBはMySQL Ver 8.0.41-0をgorpでアクセス、JavaScriptはあんまり使いたくない、かんたんなものは別として複雑なもの処理量の多いものはgoのWebAssemblyで済ませたい。",
	"ESP32":   "あなたは ESP32 のエキスパート。開発にはArduinoIDEを使っています。",
	"Arduino": "あなたは Arduino のエキスパート。開発にはArduinoIDEを使っています。",
	"OPi":     "あなたは Orange Pi Zero 3のエキスパート。OSはUbuntu系で開発にはGo 1.23.1 + Cgo + WiringOP を使っています。",
}

type Venderinf struct {
	EvAPI  string
	Apikey string
	Url    string
}

var venderlist map[string]Venderinf = map[string]Venderinf{
	"Goodle":    {EvAPI: "GOOGLE_API_KEY", Url: "https://generativelanguage.googleapis.com/v1beta/models/"},
	"Anthropic": {EvAPI: "CLAUDE_API_KEY", Url: "https://api.anthropic.com/v1/messages"},
	// "DeepSeek":  {EvAPI: "DEEPSEEK_API_KEY", Url: "https://api.deepseek.com"},
	// "DeepSeek":  {EvAPI: "DEEPSEEK_API_KEY", Url: "https://api.deepseek.com/v1"},
	"DeepSeek":  {EvAPI: "DEEPSEEK_API_KEY", Url: "https://api.deepseek.com/v1/chat/completions"},
	"OpenAI":    {EvAPI: "OPENAI_API_KEY", Url: "https://api.openai.com/v1/chat/completions"},
}

var modellist map[string]Modeltype = map[string]Modeltype{
	"gemini-2.0-flash":           {Model: "gemini", Vendor: "Goodle"},
	"claude-3-7-sonnet-20250219": {Model: "claude", Vendor: "Anthropic"},
	"claude-3-5-haiku-20241022":  {Model: "claude", Vendor: "Anthropic"},
	"deepseek-chat":              {Model: "deepseek", Vendor: "DeepSeek"},
	"deepseek-code":              {Model: "deepseek", Vendor: "DeepSeek"},
	"deepseek-reasoner":          {Model: "deepseek", Vendor: "DeepSeek"},
	"gpt-4o-mini-2024-07-18":     {Model: "openai", Vendor: "OpenAI"},
	"o3-mini-2025-01-31":         {Model: "openai", Vendor: "OpenAI"},
}

func HandlerDschat(
	w http.ResponseWriter,
	r *http.Request,
) {
	/*
		    // リクエストメソッドを確認
			fmt.Println("Method:", r.Method)

			// 明示的にParseFormを呼び出す
			err := r.ParseForm()
			if err != nil {
				fmt.Println("ParseForm error:", err)
				http.Error(w, "フォーム解析エラー", http.StatusBadRequest)
				return
			}

			// すべてのフォームデータをダンプ
			fmt.Println("Form data:", r.Form)

			// actionの値を取得
			action := r.FormValue("action")
			fmt.Println("Action value:", action)

			// POSTデータのみを確認
			fmt.Println("PostForm data:", r.PostForm)

			// 以下、通常の処理
	*/

	// 1 ページあたりのレコード数
	const pageSize = 10

	history = make([]qah, 0)

	// ページ番号を取得 (デフォルトは 1)
	pageStr := r.FormValue("action")
	// pageStr := action
	// pageStr := r.URL.Query().Get("action")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	type Top struct {
		Qa           Qa_recordsDB
		Qalist       []Qa_recordsDB
		SIselected   string
		Modellist    []string // AIモデル名のリスト
		Target       string   // 全文検索キーワード
		HasNext      bool
		HasPrevious  bool
		NextPage     int
		PreviousPage int
		Stmp         string
	}
	top := Top{}
	for k := range modellist {
		top.Modellist = append(top.Modellist, k)
	}

	// フォームデータの処理
	top.SIselected = r.FormValue("system")
	if top.SIselected == "" {
		top.SIselected = "Go"
	}
	top.Qa.System = systemlist[top.SIselected]
	question := r.FormValue("question")
	top.Qa.Modelname = r.FormValue("modelname")
	if top.Qa.Modelname == "" {
		top.Qa.Modelname = "deepseek-chat"
	}
	dsmt := r.FormValue("maxtokens")
	top.Qa.Maxtokens, _ = strconv.Atoi(dsmt)
	if top.Qa.Maxtokens == 0 {
		top.Qa.Maxtokens = 2000
	}
	top.Stmp = r.FormValue("temperature")
	if top.Stmp == "" {
		top.Stmp = "0.2"
	}
	top.Qa.Temperature, _ = strconv.ParseFloat(top.Stmp, 64)

	top.Target = r.FormValue("target")

	for i := 9; i >= 0; i-- {
		it := r.FormValue(fmt.Sprintf("checkbox%d", i))
		if it == "on" {
			history = append(history,
				qah{Sid: r.FormValue(fmt.Sprintf("id%d", i)),
					Question: r.FormValue(fmt.Sprintf("question%d", i)),
					Answer:   r.FormValue(fmt.Sprintf("answer%d", i)),
					Model:    r.FormValue(fmt.Sprintf("modelname%d", i))})
		}

	}

	// 質問がある場合は質問のリクエストを送信
	if question != "" {
		top.Qa.Question = question
		apiKey := venderlist[modellist[top.Qa.Modelname].Vendor].Apikey
		url := venderlist[modellist[top.Qa.Modelname].Vendor].Url
		model := modellist[top.Qa.Modelname].Model
		switch model {
		case "deepseek":
			err = askOpenAI(&top.Qa, history, url, apiKey)
		case "claude":
			err = askClaude(&top.Qa, history, url, apiKey)
		case "gemini":
			err = askGemini(&top.Qa, history, url, apiKey)
		case "openai":
			err = askOpenAI(&top.Qa, history, url, apiKey)
		default:
			err = fmt.Errorf("invalid modelname: %s", top.Qa.Modelname)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}
		if err != nil {
			err = fmt.Errorf("API error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}

		qbu := top.Qa.Question
		top.Qa.Question += "\nwith ID:"
		for i := 0; i < len(history); i++ {
			if i != 0 {
				top.Qa.Question += ", "
			}
			top.Qa.Question += history[i].Sid
		}

		err = srdblib.Dbmap.Insert(&top.Qa)
		top.Qa.Question = qbu
		if err != nil {
			err = fmt.Errorf("Insert() Database error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}
		// log.Printf("qadb=%+v\n", top.Qa)
	}

	// データベースからデータを取得
	offset := (page - 1) * pageSize
	var intf []interface{}
	sqlst := ""
	if top.Target == "" {
		sqlst = "SELECT * FROM qa_records ORDER BY id DESC LIMIT ? OFFSET ? "
		intf, err = srdblib.Dbmap.Select(&Qa_recordsDB{}, sqlst, pageSize, offset)
	} else {
		sqlst = "SELECT * FROM qa_records "
		sqlst += " WHERE MATCH(question, answer) AGAINST( ? IN BOOLEAN MODE) "
		sqlst += " ORDER BY id DESC LIMIT ? OFFSET ? "
		intf, err = srdblib.Dbmap.Select(&Qa_recordsDB{}, sqlst, top.Target, pageSize, offset)
	}
	if err != nil {
		err = fmt.Errorf("Select() Database error. err = %w", err)
		log.Printf("%s\n", err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	top.Qalist = make([]Qa_recordsDB, 0)
	for _, v := range intf {
		top.Qalist = append(top.Qalist, *v.(*Qa_recordsDB))
	}

	// 「次へ」と「前へ」ボタンの表示を制御
	if page > 1 {
		top.HasPrevious = true
		top.PreviousPage = page - 1
	}

	// 総レコード数を取得
	var totalRecords int64
	if top.Target == "" {
		err = srdblib.Dbmap.SelectOne(&totalRecords, "SELECT COUNT(*) FROM qa_records")
	} else {
		err = srdblib.Dbmap.SelectOne(&totalRecords,
			"SELECT COUNT(*) FROM qa_records WHERE MATCH(question, answer) AGAINST( ? IN BOOLEAN MODE)",
			top.Target)
	}
	if err != nil {
		err = fmt.Errorf("SelectOne() Database error. err = %w", err)
		log.Printf("%s\n", err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	// 「次へ」ボタンを表示するかどうかを判定
	if offset+pageSize < int(totalRecords) {
		top.HasNext = true
		top.NextPage = page + 1
	}

	// テンプレートをパースして実行
	funcMap := template.FuncMap{
		"add":           func(a, b int) int { return a + b },
		"TimeToStringY": func(t time.Time) string { return t.Format("06-01-02 15:04") },
		"sprintfResponsetime":       func(format string, n int64) string { return fmt.Sprintf(format, float32(n)/1000.0) },
		"colorOfModel":			  func(model string) template.CSS {
			if strings.Contains(model, "gemini") {
				// return "darkblue"
				return template.CSS("hsl(0,100%,50%)")
			}
			if strings.Contains(model, "claude") {
				// return "magenta"
				return template.CSS("hsl(90, 100%, 20%)")
			}
			if strings.Contains(model, "deepseek") {
				// return "darkgreen"
				return template.CSS("hsl(180, 100%, 30%)")
			}
			// return "tomato"
				return template.CSS("hsl(270, 100%, 50%)")
		},
	}

	tpl := template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/dschat.html"))

	if err := tpl.ExecuteTemplate(w, "dschat.html", top); err != nil {
		err = fmt.Errorf("tpl.ExceuteTemplate(w,\"dschat.html\", top) err=%s", err.Error())
		log.Printf("err=%s\n", err.Error())
		w.Write([]byte(err.Error()))
	}
}
