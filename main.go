package main

import (
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var cache map[string]string

func init() {
	// .envファイルを読み込む
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// キャッシュを初期化
	cache = make(map[string]string)
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
		if answer, ok := cache[question]; ok {
			c.JSON(http.StatusOK, gin.H{"answer": answer})
			return
		}

		// DeepSeek APIにリクエストを送信
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		answer, err := askDeepSeek(question, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// キャッシュに保存
		cache[question] = answer
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
		cache = make(map[string]string)
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

// キャッシュをファイルに保存
func saveCache() {
	data, err := json.Marshal(cache)
	if err != nil {
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
