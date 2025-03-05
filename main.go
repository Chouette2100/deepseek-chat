package main

import (
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// var cache map[string]string
type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Ts       string `json:"ts"`
}

var cache []QA

func init() {
	// .envファイルを読み込む
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// キャッシュを初期化
	// cache = make(map[string]string)

	// キャッシュファイル cache/QA.json を cache/yymmdd_hhmmss.json に変更
	now := time.Now()
	newFilename := fmt.Sprintf("cache/%s.json", now.Format("20060102_150405"))
	oldFilename := "cache/QA.json"
	err = os.Rename(oldFilename, newFilename)
	if err != nil {
		fmt.Println("Error renaming cache file:", err)
		// ファイルが存在しない場合でも、処理を続けるためにエラーを握りつぶす
	}

	// キャッシュをクリアする
	cache = make([]QA, 0)
	loadCache()
}

func main() {
	r := gin.Default()

	// 静的ファイルの提供
	r.LoadHTMLGlob("templates/*")

	// トップページ
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 質問を送信
	r.POST("/ask", func(c *gin.Context) {
		question := c.PostForm("question")

		// キャッシュに存在する場合はキャッシュから返す
		// if answer, ok := cache[question]; ok {
		// 	c.JSON(http.StatusOK, gin.H{"answer": answer})
		// 	return
		// }

		// DeepSeek APIにリクエストを送信
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		answer, err := askDeepSeek(question, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// キャッシュに保存
		// cache[question] = answer
		cache = append(cache, QA{question, answer, time.Now().Format("2006-01-02 15:04:05")})
		saveCache()

		c.JSON(http.StatusOK, gin.H{"answer": answer})
	})

	// キャッシュをファイルに保存
	r.POST("/save", func(c *gin.Context) {
		saveCache()
		c.JSON(http.StatusOK, gin.H{"message": "Cache saved"})
	})

	// キャッシュをクリア
	r.POST("/clear", func(c *gin.Context) {
		// cache = make(map[string]string)
		cache = make([]QA, 0)
		saveCache()
		c.JSON(http.StatusOK, gin.H{"message": "Cache cleared"})
	})

	r.Run(":8080")
}

/*
// DeepSeek APIに質問を送信
func askDeepSeek(question, apiKey string) (string, error) {
	// ここにDeepSeek APIへのリクエストを実装
	// 例: POSTリクエストを送信し、レスポンスを取得
	// 実際のAPIエンドポイントとリクエスト/レスポンスの形式に合わせて実装してください
	return "This is a mock response for: " + question, nil
}
*/

func loadCache() (err error) {
	filename := "cache/QA.json"
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	cache = make([]QA, 0)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cache)
	if err != nil {
		return
	}

	return
}

func saveCache() error {
	filename := "cache/QA.json"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ") // インデントを設定（オプション）
	return encoder.Encode(cache)
}

/*
// キャッシュをファイルに保存
func saveCache() {
	data, err := json.Marshal(cache)
	if (err != nil) {
		fmt.Println("Error marshalling cache:", err)
		return
	}
	// err = ioutil.WriteFile("cache/cache.json", data, 0644)
	err = os.WriteFile("cache/cache.json", data, 0644)
	if err != nil {
		fmt.Println("Error writing cache to file:", err)
	}
}

// キャッシュをファイルから読み込む
func loadCache() {
	// data, err := ioutil.ReadFile("cache/cache.json")
	data, err := os.ReadFile("cache/cache.json")
	if err != nil {
		fmt.Println("Error reading cache file:", err)
		return
	}
	err = json.Unmarshal(data, &cache)
	if err != nil {
		fmt.Println("Error unmarshalling cache:", err)
	}
}
*/
