package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	// "os"
	"strings"
	"time"
)

// リクエスト構造体
type MoonshotRequest struct {
	Model               string            `json:"model"`
	Messages            []MoonshotMessage `json:"messages"`
	MaxCompletionTokens int               `json:"max_completion_tokens"`
	Stream              bool              `json:"stream,omitempty"`
}

type MoonshotMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type MoonshotContentPart struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *MoonshotImageURL `json:"image_url,omitempty"`
}

type MoonshotImageURL struct {
	URL string `json:"url"`
}

type MoonshotFileObject struct {
	ID            string `json:"id"`
	Filename      string `json:"filename"`
	MimeType      string `json:"mime_type"`
	SizeBytes     int64  `json:"size_bytes"`
	CreatedAt     string `json:"created_at"`
	ExtractStatus string `json:"extract_status"`
	Purpose       string `json:"purpose"`
}

// const apiURL = "https://api.openai.com/v1/chat/completions"

func askMoonshot(
	qa *Qa_recordsDB,
	history []qah,
	url string, // APIエンドポイント
	apiKey string,
) (
	err error,
) {

	// リクエストボディ
	var msgs []MoonshotMessage

	if qa.System != "" {
		msgs = append(msgs, MoonshotMessage{Role: "system", Content: qa.System})
	}

	// 添付ファイルがある場合、ドキュメントなら内容を system メッセージとして追加する
	baseURL := strings.TrimSuffix(url, "/chat/completions")
	if qa.AttachmentName != "" && !strings.HasPrefix(qa.AttachmentMime, "image/") {
		var fileContent string
		fileContent, err = extractAttachmentContent(qa, apiKey, baseURL)
		if err != nil {
			err = fmt.Errorf("添付ファイルの処理に失敗しました: %w", err)
			log.Printf("%s", err)
			return
		}
		if fileContent != "" {
			msgs = append(msgs, MoonshotMessage{
				Role:    "system",
				Content: fmt.Sprintf("以下は添付ファイル「%s」の内容です。\n\n%s", qa.AttachmentName, fileContent),
			})
		}
	}

	// Q&Aの履歴を追加
	for i := range history {
		whotoldme := ""
		whodiditell := ""
		if history[i].Model != qa.Modelname {
			whotoldme = history[i].Model + " は答えました:\n"
			whodiditell = history[i].Model + " に聞きました:\n"
		}
		msgs = append(msgs, MoonshotMessage{Role: "user", Content: whodiditell + history[i].Question})
		msgs = append(msgs, MoonshotMessage{Role: "assistant", Content: whotoldme + history[i].Answer})
	}

	// ユーザーの質問を追加（画像添付がある場合は content を配列にする）
	if qa.AttachmentName != "" && strings.HasPrefix(qa.AttachmentMime, "image/") {
		parts := []MoonshotContentPart{
			{Type: "text", Text: qa.Question},
			{
				Type: "image_url",
				ImageURL: &MoonshotImageURL{
					URL: fmt.Sprintf("data:%s;base64,%s", qa.AttachmentMime, qa.AttachmentData),
				},
			},
		}
		msgs = append(msgs, MoonshotMessage{Role: "user", Content: parts})
	} else {
		msgs = append(msgs, MoonshotMessage{Role: "user", Content: qa.Question})
	}

	// リクエストの作成
	request := MoonshotRequest{
		Model:               qa.Modelname,
		Messages:            msgs,
		MaxCompletionTokens: qa.Maxtokens,
		Stream:              false,
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
		printJSON(body)
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

// extractAttachmentContent はドキュメント系の添付ファイルを Moonshot Files API にアップロードし、
// 抽出されたテキスト内容を返す。
func extractAttachmentContent(qa *Qa_recordsDB, apiKey string, baseURL string) (string, error) {
	if qa.AttachmentData == "" {
		return "", nil
	}
	// Base64 デコードして元のファイルデータを取得
	data, err := base64.StdEncoding.DecodeString(qa.AttachmentData)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	// Moonshot Files API にアップロードして内容を抽出
	fileObj, err := uploadFileToMoonshot(apiKey, baseURL, data, qa.AttachmentName, "file-extract")
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	// 抽出処理が完了するまで少し待つ
	time.Sleep(500 * time.Millisecond)
	content, err := getFileContentFromMoonshot(apiKey, baseURL, fileObj.ID)
	if err != nil {
		return "", fmt.Errorf("get content failed: %w", err)
	}
	return content, nil
}

// uploadFileToMoonshot はファイルを Moonshot Files API にアップロードし、file_id を返す。
func uploadFileToMoonshot(apiKey string, baseURL string, data []byte, filename string, purpose string) (*MoonshotFileObject, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("CreateFormFile failed: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("writing file data failed: %w", err)
	}
	if err := writer.WriteField("purpose", purpose); err != nil {
		return nil, fmt.Errorf("writing purpose failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer failed: %w", err)
	}

	url := baseURL + "/files"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading upload response failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var fileObj MoonshotFileObject
	if err := json.Unmarshal(respBody, &fileObj); err != nil {
		return nil, fmt.Errorf("parsing upload response failed: %w", err)
	}
	return &fileObj, nil
}

// getFileContentFromMoonshot は file_id に対応するファイルの抽出内容を取得する。
func getFileContentFromMoonshot(apiKey string, baseURL string, fileID string) (string, error) {
	url := baseURL + "/files/" + fileID + "/content"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("http.NewRequest failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get content request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading file content failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get content failed (status %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
