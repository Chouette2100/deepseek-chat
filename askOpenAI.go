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
type OpenaiRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
}
type OpenaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// レスポンス構造体
type OpenaiResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int64       `json:"created"` // UNIXタイムスタンプは int64 中にした方が無難
	Model             string      `json:"model"`
	Choices           []Choice    `json:"choices"`
	Usage             OaiResUsage `json:"usage"`
	ServiceTier       string      `json:"service_tier"`
	SystemFingerprint string      `json:"system_fingerprint"`
	ErrorMsg          string      `json:"error_msg"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      OaiResMessage `json:"message"`
	Logprobs     interface{}   `json:"logprobs"` // 具体的な型が分かる場合は指定
	FinishReason string        `json:"finish_reason"`
}

type OaiResMessage struct {
	Role        string        `json:"role"`
	Content     string        `json:"content"`
	Refusal     interface{}   `json:"refusal"`     // 具体的な型が分かる場合は指定
	Annotations []interface{} `json:"annotations"` // 具体的な型が分かる場合は指定
}

type OaiResUsage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
	AudioTokens  int `json:"audio_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

// const apiURL = "https://api.openai.com/v1/chat/completions"

func askOpenAI(
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
	request := OpenaiRequest{
		Model:       qa.Modelname,
		Messages:    msgs,
		MaxTokens:   qa.Maxtokens,
		Temperature: qa.Temperature,
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
