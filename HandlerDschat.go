// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Chouette2100/srdblib/v2"
)

func HandlerDschat(
	w http.ResponseWriter,
	r *http.Request,
) {
	// 1 ページあたりのレコード数
	const pageSize = 10

	// ページ番号を取得 (デフォルトは 1)
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	type Top struct {
		Qa           Qa_recordsDB
		Qalist       []Qa_recordsDB
		HasNext      bool
		HasPrevious  bool
		NextPage     int
		PreviousPage int
		Stmp		 string
	}
	top := Top{}

	// フォームデータの処理
	question := r.FormValue("question")
	top.Qa.Modelname = r.FormValue("modelname")
	if top.Qa.Modelname == "" {
		top.Qa.Modelname = "deepseek-chat"
	}
	dsmt := r.FormValue("maxtokens")
	top.Qa.Maxtokens, _ = strconv.Atoi(dsmt)
	if top.Qa.Maxtokens == 0 {
		top.Qa.Maxtokens = 1000
	}
	top.Stmp = r.FormValue("temperature")
	if top.Stmp == ""  {
		top.Stmp = "0.2"
	}
	top.Qa.Temperature, _ = strconv.ParseFloat(top.Stmp, 64)	

	// 質問がある場合は DeepSeek API にリクエストを送信
	if question != "" {
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		top.Qa.Question = question
		top.Qa.Answer, top.Qa.Timestamp, top.Qa.Responsetime, err =
			askDeepSeek(top.Qa, apiKey)
		if err != nil {
			err = fmt.Errorf("DeepSeek API error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}

		err = srdblib.Dbmap.Insert(&top.Qa)
		if err != nil {
			err = fmt.Errorf("Insert() Database error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}
		log.Printf("qadb=%+v\n", top.Qa)
	}

	// データベースからデータを取得
	offset := (page - 1) * pageSize
	sqlst := fmt.Sprintf("SELECT * FROM qa_records ORDER BY id DESC LIMIT %d OFFSET %d", pageSize, offset)
	var intf []interface{}
	intf, err = srdblib.Dbmap.Select(&Qa_recordsDB{}, sqlst)
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
	err = srdblib.Dbmap.SelectOne(&totalRecords, "SELECT COUNT(*) FROM qa_records")
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
		"TimeToStringY": func(t time.Time) string { return t.Format("06-01-02 15:04") },
	}

	tpl := template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/dschat.html"))

	if err := tpl.ExecuteTemplate(w, "dschat.html", top); err != nil {
		err = fmt.Errorf("tpl.ExceuteTemplate(w,\"dschat.html\", top) err=%s", err.Error())
		log.Printf("err=%s\n", err.Error())
		w.Write([]byte(err.Error()))
	}
}
