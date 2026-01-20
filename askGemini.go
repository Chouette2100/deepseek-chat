package main

import (
	// "bytes"
	// "encoding/json"
	"fmt"
	// "io"
	"log"
	// "net/http"
	"time"

	"google.golang.org/genai"
)

/*
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
*/

// Geminiに聞く！
// 参考 https://ai.google.dev/gemini-api/docs/quickstart?hl=ja#go
func askGemini(
	qa *Qa_recordsDB,
	history []qah,
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

	// SystemInstruction
	si := &genai.Content{
		Parts: []*genai.Part{
			{
				Text: qa.System,
			},
		},
	}

	log.Printf("history=%v\n", history)

	uq := make([]*genai.Content, 0)

	// Q&Aの履歴を追加
	for i := 0; i < len(history); i++ {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		content := genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				{
					Text: whodiditell + history[i].Question,
				},
			},
		}
		uq = append(uq, &content)
		content = genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{
					Text: whotoldme + history[i].Answer,
				},
			},
		}
		uq = append(uq, &content)
	}

	// ユーザーの質問を追加
	content := genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{
				Text: qa.Question,
			},
		},
	}
	uq = append(uq, &content)

	log.Printf("uq: %s\n", spewConfig.Sdump(uq))

	qa.Timestamp = time.Now()

	result, err := ExecQuery(
		apiKey,
		qa.Modelname,
		si,
		uq,
		float32(qa.Temperature),
		qa.Maxtokens,
	)

	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()

	// 1. candidates - content - parts の text を取得
	if result == nil {
		err = fmt.Errorf("result is nil")
		return
	}
	if len(result.Candidates) == 0 {
		err = fmt.Errorf("candidates not found")
		log.Printf("result: %s\n", spewConfig.Sdump(result))
		return
	}
	candidates := result.Candidates
	content = *candidates[0].Content
	parts := content.Parts
	log.Printf("parts=%+v\n", parts)
	text := ""
	for _, part := range parts {
		text += part.Text
	}
	log.Printf("Text: %s", text)
	qa.Answer = text

	// 2. usageMetadata の値を取得
	log.Printf("Prompt Token Count: %d", result.UsageMetadata.PromptTokenCount)
	log.Printf("Candidates Token Count: %d", result.UsageMetadata.CandidatesTokenCount)
	log.Printf("Total Token Count: %d", result.UsageMetadata.TotalTokenCount)
	qa.Itokens = int(result.UsageMetadata.PromptTokenCount)
	qa.Otokens = int(result.UsageMetadata.CandidatesTokenCount)

	return
}
