package main
import (
	"bytes"
	"encoding/json"
	"log"
)
// JSONデータを整形して出力
func printJSON(jsonData []byte) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, jsonData, "", "  ")
	if err != nil {
		log.Println("JSONの整形に失敗しました:", err)
		return
	}

	// 整形されたJSONを出力
	log.Println(prettyJSON.String())
}