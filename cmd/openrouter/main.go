package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Chouette2100/srdblib/v3"
	"github.com/go-gorp/gorp"
	"gopkg.in/yaml.v2"
)

const version = "openrouter-demo"

// ---- same data model as existing app ----
type Qa_recordsDB struct {
	Id             int       `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Responsetime   int64     `json:"responsetime"`
	Modelname      string    `json:"model_name"`
	Maxtokens      int       `json:"maxtokens"`
	Temperature    float64   `json:"temperature"`
	System         string    `json:"system"`
	Question       string    `json:"question"`
	Answer         string    `json:"answer"`
	Itokens        int       `json:"itokens"`
	Otokens        int       `json:"otokens"`
	StopReason     string    `json:"stop_reason"`
	AttachmentName string    `json:"attachment_name"`
	AttachmentMime string    `json:"attachment_mime"`
	AttachmentData string    `json:"attachment_data"`
}

type qah struct {
	Sid      string
	Model    string
	Question string
	Answer   string
}

type Modeltype struct {
	Model                  string
	Vendor                 string
	Comment                string
	UseMaxCompletionTokens bool `yaml:"use_max_completion_tokens"`
}

type Venderinf struct {
	EvAPI  string `yaml:"evapi"`
	Apikey string `yaml:"apikey"`
	Url    string `yaml:"url"`
}

type Config struct {
	Systemlist yaml.MapSlice        `yaml:"systemlist"`
	Venderlist map[string]Venderinf `yaml:"venderlist"`
	Modellist  yaml.MapSlice        `yaml:"modellist"`
	EncKey     string               `yaml:"enckey"`
}

var (
	db          *sql.DB
	dbmap       *gorp.DbMap
	systemlist  map[string]string
	systemkeys  []string
	modellist   map[string]Modeltype
	modelkeys   []string
	jwtKey      = []byte("your_secret_key")
	verifyCodes sync.Map
)

func readConfigFile(filePath string) ([]byte, error) {
	if strings.Contains(filepath.Base(filePath), ".enc.") {
		if os.Getenv("SOPS_AGE_KEY_FILE") == "" {
			return nil, fmt.Errorf("SOPS_AGE_KEY_FILE is empty")
		}
		content, err := exec.Command("sops", "--decrypt", filePath).Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("sops --decrypt failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
			}
			return nil, fmt.Errorf("sops --decrypt failed: %w", err)
		}
		return content, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}
	return content, nil
}

func applySystemList(items yaml.MapSlice) error {
	systemlist = make(map[string]string)
	systemkeys = make([]string, 0, len(items))
	for _, item := range items {
		key, ok := item.Key.(string)
		if !ok {
			return fmt.Errorf("systemlist key is not a string: %#v", item.Key)
		}
		val, ok := item.Value.(string)
		if !ok {
			return fmt.Errorf("systemlist value for %q is not a string: %#v", key, item.Value)
		}
		systemlist[key] = val
		systemkeys = append(systemkeys, key)
	}
	return nil
}

func loadConfig(filePath string) error {
	content, err := readConfigFile(filePath)
	if err != nil {
		return err
	}
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("yaml.Unmarshal config: %w", err)
	}
	if len(cfg.Systemlist) > 0 {
		if err := applySystemList(cfg.Systemlist); err != nil {
			return err
		}
	}
	modellist = make(map[string]Modeltype)
	modelkeys = make([]string, 0)
	if len(cfg.Modellist) > 0 {
		for _, item := range cfg.Modellist {
			key := item.Key.(string)
			valByte, _ := yaml.Marshal(item.Value)
			var mt Modeltype
			if err := yaml.Unmarshal(valByte, &mt); err != nil {
				return fmt.Errorf("yaml.Unmarshal model entry %q: %w", key, err)
			}
			mt.Model = key
			modellist[key] = mt
			modelkeys = append(modelkeys, key)
		}
	}
	if len(modelkeys) == 0 {
		modellist = map[string]Modeltype{
			"openai/gpt-4o-mini":          {Model: "openai/gpt-4o-mini", Vendor: "openrouter", Comment: "軽量", UseMaxCompletionTokens: false},
			"openai/gpt-4.1-mini":         {Model: "openai/gpt-4.1-mini", Vendor: "openrouter", Comment: "バランス", UseMaxCompletionTokens: false},
			"anthropic/claude-3.5-sonnet": {Model: "anthropic/claude-3.5-sonnet", Vendor: "openrouter", Comment: "長文推奨", UseMaxCompletionTokens: false},
			"google/gemini-2.5-flash":     {Model: "google/gemini-2.5-flash", Vendor: "openrouter", Comment: "高速", UseMaxCompletionTokens: false},
			"deepseek/deepseek-v3":        {Model: "deepseek/deepseek-v3", Vendor: "openrouter", Comment: "DeepSeek", UseMaxCompletionTokens: false},
		}
		modelkeys = []string{"openai/gpt-4o-mini", "openai/gpt-4.1-mini", "anthropic/claude-3.5-sonnet", "google/gemini-2.5-flash", "deepseek/deepseek-v3"}
	}
	if len(systemkeys) == 0 {
		if err := applySystemList(yaml.MapSlice{
			{Key: "default", Value: "You are a careful and helpful assistant."},
			{Key: "tech", Value: "You are a senior software engineer. Provide accurate and concise answers."},
			{Key: "research", Value: "You are a research assistant. Explain clearly and cite assumptions."},
		}); err != nil {
			return err
		}
	}
	return nil
}

func setupDB() error {
	dbInstance, dbconfig, err := srdblib.OpenDb("DBConfig.enc.yml")
	if err != nil {
		return fmt.Errorf("OpenDb failed: %w", err)
	}
	if dbconfig.UseSSH {
		defer srdblib.Dialer.Close()
	}
	db = dbInstance
	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	dbmap = &gorp.DbMap{Db: db, Dialect: dial, ExpandSliceArgs: true}
	dbmap.AddTableWithName(Qa_recordsDB{}, "qa_records").SetKeys(true, "Id")
	return nil
}

// ---- OpenRouter API types ----
type OpenRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	ID      string             `json:"id"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage"`
	Error   *OpenRouterError   `json:"error,omitempty"`
}

type OpenRouterChoice struct {
	Index        int                  `json:"index"`
	Message      OpenRouterMessageOut `json:"message"`
	FinishReason string               `json:"finish_reason"`
}

type OpenRouterMessageOut struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenRouterError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func askOpenRouter(qa *Qa_recordsDB, history []qah, model, apiKey string) error {
	startedAt := time.Now()
	var msgs []OpenRouterMessage
	if qa.System != "" {
		msgs = append(msgs, OpenRouterMessage{Role: "system", Content: qa.System})
	}
	for _, h := range history {
		if h.Model != "" && h.Model != qa.Modelname {
			msgs = append(msgs, OpenRouterMessage{Role: "user", Content: "Previous model: " + h.Model + "\n" + h.Question})
			msgs = append(msgs, OpenRouterMessage{Role: "assistant", Content: h.Answer})
			continue
		}
		msgs = append(msgs, OpenRouterMessage{Role: "user", Content: h.Question})
		msgs = append(msgs, OpenRouterMessage{Role: "assistant", Content: h.Answer})
	}
	msgs = append(msgs, OpenRouterMessage{Role: "user", Content: qa.Question})

	body, err := json.Marshal(OpenRouterRequest{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   qa.Maxtokens,
		Temperature: qa.Temperature,
	})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://localhost")
	req.Header.Set("X-Title", "OpenRouter Demo")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do: %w", err)
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OpenRouter API error status=%d body=%s", resp.StatusCode, string(resBody))
	}

	var result OpenRouterResponse
	if err := json.Unmarshal(resBody, &result); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}
	if result.Error != nil {
		return fmt.Errorf("OpenRouter error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return fmt.Errorf("OpenRouter: empty choices")
	}

	qa.Answer = result.Choices[0].Message.Content
	qa.Itokens = result.Usage.PromptTokens
	qa.Otokens = result.Usage.CompletionTokens
	qa.StopReason = result.Choices[0].FinishReason
	qa.Timestamp = startedAt
	qa.Responsetime = time.Since(startedAt).Milliseconds()
	return nil
}

// ---- web UI ----

type templateData struct {
	Qa          Qa_recordsDB
	Qalist      []Qa_recordsDB
	SIselected  string
	Modellist   []Modeltype
	Target      string
	HasNext     bool
	HasPrevious bool
	Stmp        string
	Systemkeys  []string
	StartID     int
}

func openRouterHandler(w http.ResponseWriter, r *http.Request) {
	const pageSize = 20
	var history []qah
	_ = r.ParseMultipartForm(32 << 20)

	currentID, _ := strconv.Atoi(r.FormValue("startid"))
	action := r.FormValue("action")
	maxID := int64(0)
	if err := dbmap.SelectOne(&maxID, "SELECT MAX(id) FROM qa_records"); err != nil {
		maxID = 0
	}
	if currentID == 0 {
		currentID = int(maxID)
	}
	switch action {
	case "next":
		currentID -= pageSize
	case "previous":
		currentID += pageSize
	}
	if currentID < 1 {
		currentID = 1
	}
	if currentID > int(maxID) {
		currentID = int(maxID)
	}

	top := templateData{StartID: currentID, Stmp: r.FormValue("temperature")}
	if top.Stmp == "" {
		top.Stmp = "0.2"
	}
	top.SIselected = r.FormValue("system")
	if top.SIselected == "" {
		top.SIselected = systemkeys[0]
	}
	top.Qa.System = systemlist[top.SIselected]
	top.Qa.Modelname = r.FormValue("modelname")
	if top.Qa.Modelname == "" {
		top.Qa.Modelname = "openai/gpt-4o-mini"
	}
	if dsmt := r.FormValue("maxtokens"); dsmt != "" {
		top.Qa.Maxtokens, _ = strconv.Atoi(dsmt)
	}
	if top.Qa.Maxtokens == 0 {
		top.Qa.Maxtokens = 2000
	}
	top.Qa.Temperature, _ = strconv.ParseFloat(top.Stmp, 64)
	top.Target = r.FormValue("target")

	for _, key := range modelkeys {
		mt := modellist[key]
		mt.Model = key
		top.Modellist = append(top.Modellist, mt)
	}
	top.Systemkeys = systemkeys

	file, header, ferr := r.FormFile("attachment")
	if ferr == nil && file != nil {
		defer file.Close()
		data, err := io.ReadAll(file)
		if err == nil {
			top.Qa.AttachmentName = header.Filename
			top.Qa.AttachmentMime = header.Header.Get("Content-Type")
			if top.Qa.AttachmentMime == "" {
				top.Qa.AttachmentMime = http.DetectContentType(data)
			}
			top.Qa.AttachmentData = base64.StdEncoding.EncodeToString(data)
		}
	}

	for i := 0; i < 20; i++ {
		if r.FormValue(fmt.Sprintf("checkbox%d", i)) == "on" {
			history = append(history, qah{
				Sid:      r.FormValue(fmt.Sprintf("id%d", i)),
				Question: r.FormValue(fmt.Sprintf("question%d", i)),
				Answer:   r.FormValue(fmt.Sprintf("answer%d", i)),
				Model:    r.FormValue(fmt.Sprintf("modelname%d", i)),
			})
		}
	}

	question := r.FormValue("question")
	if question != "" {
		top.StartID = int(maxID) + 1
		top.Qa.Question = question
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			log.Println("OPENROUTER_API_KEY is empty")
			http.Error(w, "OPENROUTER_API_KEY is not set", http.StatusUnauthorized)
			return
		}
		if err := askOpenRouter(&top.Qa, history, top.Qa.Modelname, apiKey); err != nil {
			log.Printf("askOpenRouter failed: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		top.Qa.Question = question
		if err := dbmap.Insert(&top.Qa); err != nil {
			log.Printf("dbmap.Insert failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var intf []interface{}
	var err error
	if top.Target == "" {
		intf, err = dbmap.Select(&Qa_recordsDB{}, "SELECT id, timestamp, responsetime, modelname, maxtokens, temperature, `system`, question, answer, itokens, otokens, stopreason, attachmentname, attachmentmime, attachmentdata FROM qa_records WHERE id <= ? ORDER BY id DESC LIMIT ?", top.StartID, pageSize)
	} else {
		q := "%" + top.Target + "%"
		intf, err = dbmap.Select(&Qa_recordsDB{}, "SELECT id, timestamp, responsetime, modelname, maxtokens, temperature, `system`, question, answer, itokens, otokens, stopreason, attachmentname, attachmentmime, attachmentdata FROM qa_records WHERE id <= ? AND (question LIKE ? OR answer LIKE ?) ORDER BY id DESC LIMIT ?", top.StartID, q, q, pageSize)
	}
	if err != nil {
		log.Printf("db query failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	top.Qalist = make([]Qa_recordsDB, 0, len(intf))
	for _, v := range intf {
		top.Qalist = append(top.Qalist, *v.(*Qa_recordsDB))
	}
	if top.StartID < int(maxID) {
		top.HasPrevious = true
	}
	if len(top.Qalist) > 0 {
		lastID := top.Qalist[len(top.Qalist)-1].Id
		var countBelow int64
		if top.Target == "" {
			if err := dbmap.SelectOne(&countBelow, "SELECT COUNT(*) FROM qa_records WHERE id < ?", lastID); err == nil && countBelow > 0 {
				top.HasNext = true
			}
		} else {
			q := "%" + top.Target + "%"
			if err := dbmap.SelectOne(&countBelow, "SELECT COUNT(*) FROM qa_records WHERE id < ? AND (question LIKE ? OR answer LIKE ?)", lastID, q, q); err == nil && countBelow > 0 {
				top.HasNext = true
			}
		}
	}

	funcMap := template.FuncMap{
		"add":                 func(a, b int) int { return a + b },
		"TimeToStringY":       func(t time.Time) string { return t.Format("06-01-02 15:04") },
		"sprintfResponsetime": func(format string, n int64) string { return fmt.Sprintf(format, float32(n)/1000.0) },
		"colorOfModel": func(model string) template.CSS {
			if strings.Contains(model, "gemini") {
				return template.CSS("hsl(0,100%,50%)")
			}
			if strings.Contains(model, "claude") {
				return template.CSS("hsl(90, 100%, 20%)")
			}
			if strings.Contains(model, "deepseek") {
				return template.CSS("hsl(180, 100%, 30%)")
			}
			if strings.Contains(model, "gpt") || strings.Contains(model, "o1") || strings.Contains(model, "o3") {
				return template.CSS("hsl(30, 100%, 50%)")
			}
			if strings.Contains(model, "kimi") || strings.Contains(model, "moonshot") {
				return template.CSS("hsl(120, 100%, 30%)")
			}
			return template.CSS("hsl(270, 100%, 50%)")
		},
	}

	tpl := template.Must(template.New("openrouter.html").Funcs(funcMap).ParseFiles("templates/openrouter.html"))
	if err := tpl.ExecuteTemplate(w, "openrouter.html", top); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	if err := loadConfig("config.enc.yml"); err != nil {
		log.Printf("loadConfig failed: %v", err)
	}
	if err := setupDB(); err != nil {
		log.Fatalf("setupDB failed: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/openrouter", openRouterHandler)
	http.HandleFunc("/", openRouterHandler)
	log.Println("OpenRouter demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
