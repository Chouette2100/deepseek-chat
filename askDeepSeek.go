package main
import (
	"bytes"
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"io"
	"net/http"
)
func askDeepSeek(question, apiKey string) (string, error) {
	// APIエンドポイント
	url := "https://api.deepseek.com/v1/chat/completions"

	// リクエストボディ
	payload := map[string]interface{}{
		"model": "deepseek-chat", // 使用するモデルを指定
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": question,
			},
		},
		// "max_tokens": 150, // 最大トークン数を指定
		"max_tokens": 1000, // 最大トークン数を指定
	}
	// JSONに変換
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %v", err)
	}

	// HTTPリクエストを作成
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// ヘッダーを設定
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// リクエストを送信
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスを読み取る
	// body, err := ioutil.ReadAll(resp.Body)
	body, err := io.ReadAll(resp.Body)
	// body, err := os.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// レスポンスをパース
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	// 回答を取得
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format")
}
