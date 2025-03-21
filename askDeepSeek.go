package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	// "io/ioutil"
	"io"
	"log"
	"net/http"
)

// func askDeepSeek(
func AskDeepSeek(
	qa *Qa_recordsDB,
	history []qah,
	url string, // APIエンドポイント
	apiKey string,
) (
	err error,
) {

	// リクエストボディ
	var msgs []map[string]string

	// Claudeで言うsystem、要するに大前提
	if qa.System != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": qa.System})
	}

	// Q&Aの履歴を追加
	for i := 0; i < len(history); i++ {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		msgs = append(msgs, map[string]string{"role": "user", "content": whodiditell + history[i].Question})
		msgs = append(msgs, map[string]string{"role": "assistant", "content": whotoldme + history[i].Answer})
	}

	// ユーザーの質問を追加
	msgs = append(msgs, map[string]string{"role": "user", "content": qa.Question})

	payload := map[string]interface{}{
		// "model": "deepseek-chat", // 使用するモデルを指定
		"model": qa.Modelname, // 使用するモデルを指定
		/*
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": qa.Question,
				},
			},
		*/
		"messages": msgs,
		// "max_tokens": 150, // 最大トークン数を指定
		"max_tokens":  qa.Maxtokens,   // 最大トークン数を指定
		"temperature": qa.Temperature, // 温度を指定
	}
	// JSONに変換
	var jsonData []byte
	jsonData, err = json.Marshal(payload)
	if err != nil {
		err = fmt.Errorf("error marshalling payload: %v", err)
		return
	}

	// printJSON(jsonData)

	// HTTPリクエストを作成
	var req *http.Request
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		err = fmt.Errorf("error creating request: %v", err)
		return
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// リクエストを送信
	qa.Timestamp = time.Now()
	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		err = fmt.Errorf("error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取る
	// body, err := ioutil.ReadAll(resp.Body)
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("error reading response body: %v", err)
		return
	}
	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()

	printJSON(body)

	res := OpenaiResponse{}
	if err = json.Unmarshal(body, &res); err != nil {
		// err = fmt.Errorf("error unmarshalling response: %v", err)
		log.Printf("error unmarshalling response: %v", err)
		// return
	}
	// log.Printf("%+v\n", res)
	log.Printf(" answer=%s\n", res.Choices[0].Message.Content)
	// qa.Answer = res.Choices[0].Message.Content
	qa.Itokens = res.Usage.PromptTokens
	qa.Otokens = res.Usage.CompletionTokens

	// レスポンスをパース
	var result map[string]interface{}
	if err = json.Unmarshal(body, &result); err != nil {
		err = fmt.Errorf("error unmarshalling response: %v", err)
		return
	}

	// 回答を取得
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if qa.Answer, ok = message["content"].(string); ok {
					return
				}
			}
		}
	}

	err = fmt.Errorf("invalid response format")
	log.Printf("%s", err)

	return
}
