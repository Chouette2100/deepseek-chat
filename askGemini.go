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

type GeminiPayload struct {
	Contents          []Contents        `json:"contents"`
	SystemInstruction SystemInstruction `json:"systemInstruction"`
	GenerationConfig  GenerationConfig  `json:"generationConfig"`
}
type Parts struct {
	Text string `json:"text"`
}
type Contents struct {
	Role  string  `json:"role"`
	Parts []Parts `json:"parts"`
}
type SystemInstruction struct {
	// Role  string  `json:"role"`
	Parts []Parts `json:"parts"`
}
type GenerationConfig struct {
	Temperature float64 `json:"temperature"`
	// TopP            float64 `json:"topP"`
	// TopK            float64 `json:"topK"`
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type GmnResponse struct {
	Candidates    []GmnResCandidates  `json:"candidates"`
	UsageMetadata GmnResUsageMetadata `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
}
type GmnResParts struct {
	Text string `json:"text"`
}
type GmnResContent struct {
	Parts []Parts `json:"parts"`
	Role  string  `json:"role"`
}
type GmnResCandidates struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	AvgLogprobs  float64 `json:"avgLogprobs"`
}
type GmnResPromptTokensDetails struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}
type GmnResCandidatesTokensDetails struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}
type GmnResUsageMetadata struct {
	PromptTokenCount        int                             `json:"promptTokenCount"`
	CandidatesTokenCount    int                             `json:"candidatesTokenCount"`
	TotalTokenCount         int                             `json:"totalTokenCount"`
	PromptTokensDetails     []PromptTokensDetails           `json:"promptTokensDetails"`
	CandidatesTokensDetails []GmnResCandidatesTokensDetails `json:"candidatesTokensDetails"`
}

func askGemini(
	qa *Qa_recordsDB,
	history []qah,
	url string, // APIエンドポイント
	apiKey string,
) (
	err error,
) {
	// apiKey := os.Getenv("GOOGLE_API_KEY") // 環境変数からAPIキーを取得
	if apiKey == "" {
		// log.Println("APIキーが設定されていません")
		err = fmt.Errorf("APIキーが設定されていません")
		return
	}

	// url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=" + apiKey
	url = url + qa.Modelname + ":generateContent?key=" + apiKey
	// url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent" //APIキーをURLに含めない
	// url := "https://generativelanguage.googleapis.com/v1beta/models/" + qa.Modelname + ":generateContent" //APIキーをURLに含めない

	log.Printf("history=%v\n", history)

	contents := make([]Contents, 0)
	// Q&Aの履歴を追加
	for i := 0; i < len(history); i++ {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		contents = append(contents, Contents{Role: "user", Parts: []Parts{{Text: whodiditell + history[i].Question}}})
		contents = append(contents, Contents{Role: "model", Parts: []Parts{{Text: whotoldme + history[i].Answer}}})
	}

	// ユーザーの質問を追加
	contents = append(contents, Contents{Role: "user", Parts: []Parts{{Text: qa.Question}}})

	payload := GeminiPayload{
		Contents: contents,
		SystemInstruction: SystemInstruction{
			Parts: []Parts{{Text: qa.System}},
		},
		GenerationConfig: GenerationConfig{
			Temperature: qa.Temperature,
			// TopP:            0.9,
			// TopK:            0,
			MaxOutputTokens: qa.Maxtokens,
		},
	}
	/*
		payload := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"role": "user",
					"parts": []map[string]interface{}{
						{
							"text": qa.Question,
						},
					},
				},
			},
			"systemInstruction": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{
							"text": qa.System,
						},
					},
				},
			},
			"generationConfig": []map[string]interface{}{
				{
					"temperature":     qa.Temperature,
					"maxOutputTokens": qa.Maxtokens,
				},
			},
		}
	*/
	var payloadBytes []byte
	payloadBytes, err = json.Marshal(&payload)
	if err != nil {
		// fmt.Println("JSONエンコードエラー:", err)
		err = fmt.Errorf("JSONエンコードエラー: %w", err)
		return
	}

	// printJSON(payloadBytes)

	var req *http.Request
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		// fmt.Println("リクエスト作成エラー:", err)
		err = fmt.Errorf("リクエスト作成エラー: %w", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("x-goog-api-key", apiKey) //APIキーをヘッダーに設定
	// req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	qa.Timestamp = time.Now()
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		// fmt.Println("リクエスト送信エラー:", err)
		err = fmt.Errorf("リクエスト送信エラー: %w", err)
		return
	}
	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("レスポンス読み込みエラー:", err)
		err = fmt.Errorf("レスポンス読み込みエラー: %w", err)
		return
	}
	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		// fmt.Println("JSONデコードエラー:", err)
		err = fmt.Errorf("JSONデコードエラー: %w", err)
		return
	}

	printJSON(body)

	// log.Println(string(body))
	// log.Printf("%+v\n", result)

	// 1. candidates - content - parts の text を取得
	if result == nil {
		err = fmt.Errorf("result is nil")
		return
	}
	if _, ok := result["candidates"]; !ok {
		err = fmt.Errorf("candidates not found")
		return
	}
	candidates := result["candidates"].([]interface{})
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	log.Printf("parts=%+v\n", parts)
	text := ""
	for _, part := range parts {
		if partMap, ok := part.(map[string]interface{}); ok {
			if textPart, ok := partMap["text"].(string); ok {
				text += textPart
			}
		}
	}
	// text := parts[0].(map[string]interface{})["text"].(string)
	fmt.Println("Text:", text)
	qa.Answer = text

	// 2. usageMetadata の値を取得
	usageMetadata := result["usageMetadata"].(map[string]interface{})
	promptTokenCount := int(usageMetadata["promptTokenCount"].(float64))
	candidatesTokenCount := int(usageMetadata["candidatesTokenCount"].(float64))
	totalTokenCount := int(usageMetadata["totalTokenCount"].(float64))
	fmt.Println("Prompt Token Count:", promptTokenCount)
	fmt.Println("Candidates Token Count:", candidatesTokenCount)
	fmt.Println("Total Token Count:", totalTokenCount)
	qa.Itokens = promptTokenCount
	qa.Otokens = candidatesTokenCount

	return
}
