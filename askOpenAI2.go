package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"io"
	"net/http"
	// "os"
	"log"
	"time"
)

// リクエスト構造体
type Openai2Request struct {
	Model               string          `json:"model"`
	Messages            []OpenaiMessage `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens"`
}

// const apiURL = "https://api.openai.com/v1/chat/completions"

func askOpenAI2(
	qa *Qa_recordsDB,
	history []qah,
	url string, // APIエンドポイント
	apiKey string,
) (
	err error,
) {

	// リクエストボディ
	var msgs []OpenaiMessage

	if qa.System != "" {
		msgs = append(msgs, OpenaiMessage{Role: "system", Content: qa.System})
	}

	// Q&Aの履歴を追加
	for i := 0; i < len(history); i++ {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		msgs = append(msgs, OpenaiMessage{Role: "user", Content: whodiditell + history[i].Question})
		msgs = append(msgs, OpenaiMessage{Role: "assistant", Content: whotoldme + history[i].Answer})
	}

	// ユーザーの質問を追加
	msgs = append(msgs, OpenaiMessage{Role: "user", Content: qa.Question})

	// リクエストの作成
	request := Openai2Request{
		Model:       qa.Modelname,
		Messages:    msgs,
		MaxCompletionTokens:   qa.Maxtokens,
	}

	// JSONに変換
	var jsonData []byte
	jsonData, err = json.Marshal(request)
	if err != nil {
		err = fmt.Errorf("on.Marshal(): JSONへの変換に失敗しました: %w", err)
		log.Printf("%s", err)
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
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// リクエストを送信
	client := &http.Client{}
	qa.Timestamp = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスを読み取る

	// body, _ := ioutil.ReadAll(resp.Body)
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("error reading response body: %v", err)
		return
	}

	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()

	printJSON(body)

	// fmt.Println(string(body))

	// レスポンスをパース
	res := OpenaiResponse{}
	if err = json.Unmarshal(body, &res); err != nil {
		// err = fmt.Errorf("error unmarshalling response: %v", err)
		log.Printf("error unmarshalling response: %v", err)
		// return
	}

	if res.ErrorMsg != "" {
		err = fmt.Errorf("API error: %s", res.ErrorMsg)
		log.Printf("%s", err.Error())
		printJSON(jsonData)
		return
	}
	if len(res.Choices) == 0 {
		err = fmt.Errorf("API error: empty response")
		log.Printf("%s", err.Error())
		printJSON(jsonData)
		return
	}
	// log.Printf("%+v\n", res)
	// log.Printf(" answer=%s\n", res.Choices[0].Message.Content)
	qa.Answer = res.Choices[0].Message.Content
	qa.Itokens = res.Usage.PromptTokens
	qa.Otokens = res.Usage.CompletionTokens

	/*
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
	*/

	return
}
