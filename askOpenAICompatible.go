package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// OpenAI 互換 API 用のリクエスト構造体
type OpenaiCompatibleRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenaiMessage `json:"messages"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	// Temperature         float64         `json:"temperature,omitempty"`
}

type OpenaiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type OpenaiContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenaiImageURL `json:"image_url,omitempty"`
}

type OpenaiImageURL struct {
	URL string `json:"url"`
}

// OpenAI 互換 API 用のレスポンス構造体
type OpenaiResponse struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []Choice      `json:"choices"`
	Usage             OaiResUsage   `json:"usage"`
	ServiceTier       string        `json:"service_tier"`
	SystemFingerprint string        `json:"system_fingerprint"`
	Error             *OaiResError  `json:"error,omitempty"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      OaiResMessage `json:"message"`
	Logprobs     any           `json:"logprobs"`
	FinishReason string        `json:"finish_reason"`
}

type OaiResMessage struct {
	Role        string `json:"role"`
	Content     string `json:"content"`
	Refusal     any    `json:"refusal"`
	Annotations []any  `json:"annotations"`
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

type OaiResError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// askOpenAICompatible は OpenAI 互換のチャット API を呼び出す統一関数です。
// DeepSeek、Claude（OpenAI 互換 beta）、Gemini（OpenAI 互換）、Moonshot、OpenAI 共通で利用できます。
func askOpenAICompatible(
	qa *Qa_recordsDB,
	history []qah,
	model string,
	url string,
	apiKey string,
	useMaxCompletionTokens bool,
) error {
	var msgs []OpenaiMessage

	if qa.System != "" {
		msgs = append(msgs, OpenaiMessage{Role: "system", Content: qa.System})
	}

	for i := range history {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		msgs = append(msgs, OpenaiMessage{Role: "user", Content: whodiditell + history[i].Question})
		msgs = append(msgs, OpenaiMessage{Role: "assistant", Content: whotoldme + history[i].Answer})
	}

	// 画像添付がある場合は OpenAI 互換の vision 形式で送信する
	if qa.AttachmentName != "" && qa.AttachmentMime != "" && strings.HasPrefix(qa.AttachmentMime, "image/") {
		parts := []OpenaiContentPart{
			{Type: "text", Text: qa.Question},
			{
				Type: "image_url",
				ImageURL: &OpenaiImageURL{
					URL: fmt.Sprintf("data:%s;base64,%s", qa.AttachmentMime, qa.AttachmentData),
				},
			},
		}
		msgs = append(msgs, OpenaiMessage{Role: "user", Content: parts})
	} else {
		msgs = append(msgs, OpenaiMessage{Role: "user", Content: qa.Question})
	}

	reqBody := OpenaiCompatibleRequest{
		Model:       model,
		Messages:    msgs,
		// Temperature: qa.Temperature,
	}
	if useMaxCompletionTokens {
		reqBody.MaxCompletionTokens = qa.Maxtokens
	} else {
		reqBody.MaxTokens = qa.Maxtokens
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("http.NewRequest(): %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	qa.Timestamp = time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("client.Do(): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll(): %w", err)
	}
	qa.Responsetime = time.Since(qa.Timestamp).Milliseconds()

	printJSON(body)

	var res OpenaiResponse
	if err = json.Unmarshal(body, &res); err != nil {
		log.Printf("error unmarshalling response: %v", err)
	}

	if res.Error != nil && res.Error.Message != "" {
		return fmt.Errorf("API error: %s (%s)", res.Error.Message, res.Error.Code)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	if len(res.Choices) == 0 {
		return fmt.Errorf("API error: empty response")
	}

	qa.Answer = res.Choices[0].Message.Content
	qa.Itokens = res.Usage.PromptTokens
	qa.Otokens = res.Usage.CompletionTokens
	qa.StopReason = res.Choices[0].FinishReason
	return nil
}
