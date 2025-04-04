package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// リクエスト構造体
type ClaudeRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature,omitempty"`
	System      string    `json:"system,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// レスポンス構造体
type ClaudeResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason"`
	StopSequence string    `json:"stop_sequence"`
	Usage        Usage     `json:"usage"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// エラー構造体
type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func askClaude(
	qa *Qa_recordsDB,
	history []qah,
	url string, // APIエンドポイント
	apiKey string,
) (
	err error,
) {

	// リクエストボディ
	var msgs []Message

	// Q&Aの履歴を追加
	for i := 0; i < len(history); i++ {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		msgs = append(msgs, Message{Role: "user", Content: whodiditell + history[i].Question})
		msgs = append(msgs, Message{Role: "assistant", Content: whotoldme + history[i].Answer})
	}

	// ユーザーの質問を追加
	msgs = append(msgs, Message{Role: "user", Content: qa.Question})

	// リクエストの作成
	request := ClaudeRequest{
		Model:       qa.Modelname,
		Messages:    msgs,
		MaxTokens:   qa.Maxtokens,
		Temperature: qa.Temperature,
		System:      qa.System,
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

	// HTTPリクエストの作成
	client := &http.Client{}
	var req *http.Request
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		err = fmt.Errorf("http.NewRequest(): リクエストの作成に失敗しました: %w", err)
		log.Printf("%s", err)
		return
	}

	// ヘッダーの設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// リクエストの送信
	qa.Timestamp = time.Now()
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		err = fmt.Errorf("client.Do(): APIリクエストの送信に失敗しました: %w", err)
		log.Printf("%s", err)
		return
	}
	defer resp.Body.Close()

	// レスポンスの読み取り
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("io.ReadAll(): レスポンスの読み取りに失敗しました: %w", err)
		log.Printf("%s", err)
		return
	}
	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()
	log.Printf("レスポンス時間: %d ms\n", qa.Responsetime)
	printJSON(body)

	// ステータスコードの確認
	if resp.StatusCode != http.StatusOK {
		// エラーレスポンスの解析
		var claudeError ClaudeError
		if err = json.Unmarshal(body, &claudeError); err == nil {
			err = fmt.Errorf("json.Unmarshal(): API エラー: %s - %s", claudeError.Type, claudeError.Message)
			log.Printf("%s", err)
			printJSON(jsonData)
			return
		} else {
			err = fmt.Errorf("json.Unmarshal(): API エラー: ステータスコード %d, レスポンス: %s", resp.StatusCode, string(body))
			log.Printf("%s", err)
			printJSON(jsonData)
			return
		}
	}

	// printJSON(body)

	// 成功レスポンスの解析
	var claudeResponse ClaudeResponse
	if err = json.Unmarshal(body, &claudeResponse); err != nil {
		err = fmt.Errorf("json.Unmarshal(): レスポンスのJSONパースに失敗しました: %w", err)
		log.Printf("%s", err)
		return
	}

	// 結果の表示
	log.Printf("停止理由: %s\n", claudeResponse.StopReason)
	log.Println("role=", claudeResponse.Role)
	qa.StopReason = claudeResponse.StopReason
	qa.Itokens = claudeResponse.Usage.InputTokens
	qa.Otokens = claudeResponse.Usage.OutputTokens
	for i, content := range claudeResponse.Content {
		if content.Type == "text" {
			if i != 0 {
				qa.Answer = qa.Answer + "\n---\n"
			}
			qa.Answer = qa.Answer + content.Text
			log.Println(content.Text)
		}
	}
	return
}
