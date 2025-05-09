// Copyright © 2025 chouette2100@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	// "net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Chouette2100/srdblib/v2"
	"github.com/golang-jwt/jwt/v4"
)

type qah struct {
	Sid      string
	Model    string
	Question string
	Answer   string
}

var history []qah

type Modeltype struct {
	Model  string
	Vendor string
}

var systemlist map[string]string = map[string]string{
	"none":    "",
	"Go":      "あなたはGoのエクスパート。環境はLinuxMint21.3、Go1.23.1、net/http＋go-template、DBはMySQL Ver 8.0.41-0をgorpでアクセス、JavaScriptはかんたんなものに使って、複雑なもの処理量の多いものはgoのWebAssemblyにしたい。",
	"ESP32":   "あなたは ESP32 のエキスパート。開発にはArduinoIDEを使っています。",
	"Arduino": "あなたは Arduino のエキスパート。開発にはArduinoIDEを使っています。",
	"OPi":     "あなたは Orange Pi Zero 3のエキスパート。OSはUbuntu系で開発にはGo 1.23.1 + Cgo + WiringOP を使っています。",
}

type Venderinf struct {
	EvAPI  string
	Apikey string
	Url    string
}

var venderlist map[string]Venderinf = map[string]Venderinf{
	"Goodle":    {EvAPI: "GOOGLE_API_KEY", Url: "https://generativelanguage.googleapis.com/v1beta/models/"},
	"Anthropic": {EvAPI: "CLAUDE_API_KEY", Url: "https://api.anthropic.com/v1/messages"},
	"DeepSeek":  {EvAPI: "DEEPSEEK_API_KEY", Url: "https://api.deepseek.com/v1/chat/completions"},
	"OpenAI":    {EvAPI: "OPENAI_API_KEY", Url: "https://api.openai.com/v1/chat/completions"},
}

var modellist map[string]Modeltype = map[string]Modeltype{
	"gemini-2.0-flash":               {Model: "gemini", Vendor: "Goodle"},
	"gemini-2.5-flash-preview-04-17": {Model: "gemini", Vendor: "Goodle"},
	"gemini-2.5-pro-preview-05-06":   {Model: "gemini", Vendor: "Goodle"},
	"claude-3-7-sonnet-20250219":     {Model: "claude", Vendor: "Anthropic"},
	"claude-3-5-haiku-20241022":      {Model: "claude", Vendor: "Anthropic"},
	"deepseek-chat":                  {Model: "openai", Vendor: "DeepSeek"},
	"deepseek-code":                  {Model: "openai", Vendor: "DeepSeek"},
	"deepseek-reasoner":              {Model: "openai", Vendor: "DeepSeek"},
	"gpt-4o-mini-2024-07-18":         {Model: "openai", Vendor: "OpenAI"},
	"o3-mini-2025-01-31":             {Model: "openai2", Vendor: "OpenAI"},
	"gpt-4o-2024-08-06":              {Model: "openai", Vendor: "OpenAI"},
}

var jwtKey = []byte("your_secret_key")
var verificationCodes = sync.Map{} // To store email verification codes temporarily

// GenerateJWT generates a JWT token for a given email
func GenerateJWT(email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   email,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// Middleware to validate JWT
func ValidateJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// tokenStr := r.Header.Get("Authorization")
		// CookieからJWTを取得
		cookie, err := r.Cookie("jwt_token")
		// if err != nil {
		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }

		tokenStr := ""
		if err == nil || cookie != nil {
			// cookieの値を取得
			tokenStr = cookie.Value
		}
		if tokenStr == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// SendVerificationCode sends a 6-digit code to the user's email
func SendVerificationCode(email string) (string, error) {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	verificationCodes.Store(email, code)
	log.Printf("Verification code for %s: %s\n", email, code)
	// Simulate email sending (replace with actual SMTP logic)
	/*
		err := smtp.SendMail("smtp.example.com:587",
			smtp.PlainAuth("", "your_email@example.com", "your_password", "smtp.example.com"),
			"your_email@example.com", []string{email},
			[]byte("Subject: Verification Code\n\nYour code is: "+code))
		return code, err
	*/
	return code, nil
}

// SignupHandler handles user signup
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		_, err := SendVerificationCode(email) // Removed unused variable `code`
		if err != nil {
			http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Verification code sent to %s", email)
		return
	}
	http.ServeFile(w, r, "templates/signup.html")
}

// VerifyCodeHandler verifies the code and allows password setup
func VerifyCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// if r.Method == http.MethodGet {
		email := r.FormValue("email")
		code := r.FormValue("code")
		storedCode, ok := verificationCodes.Load(email)
		if !ok || storedCode != code {
			http.Error(w, "Invalid code", http.StatusUnauthorized)
			return
		}
		verificationCodes.Delete(email)
		http.ServeFile(w, r, "templates/set_password.html")
		return
	}
	http.Error(w, "Invalid request", http.StatusBadRequest)
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")
		// Validate email and password with database (pseudo-code)
		// if email == "test@example.com" && password == "password" {
		if email == "iapetus@seppina.com" && password == "sfbsfbsfb78" {
			token, err := GenerateJWT(email)
			if err != nil {
				http.Error(w, "Failed to generate token", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				// Name:    "token",
				Name:    "jwt_token",
				Value:   token,
				Expires: time.Now().Add(24 * time.Hour),
				// 以下追加(deepseek-chat)
				HttpOnly: true, // JavaScriptからアクセス不可
				Secure:   true, // HTTPSのみ
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			})
			// http.Redirect(w, r, "/", http.StatusSeeOther)
			http.Redirect(w, r, "/dschat", http.StatusSeeOther)
			return
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	http.ServeFile(w, r, "templates/login.html")
}

func HandlerDschat(
	w http.ResponseWriter,
	r *http.Request,
) {
	const pageSize = 20

	history = make([]qah, 0)

	pageStr := r.FormValue("action")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	type Top struct {
		Qa           Qa_recordsDB
		Qalist       []Qa_recordsDB
		SIselected   string
		Modellist    []string
		Target       string
		HasNext      bool
		HasPrevious  bool
		NextPage     int
		PreviousPage int
		Stmp         string
	}
	top := Top{}
	for k := range modellist {
		top.Modellist = append(top.Modellist, k)
	}

	top.SIselected = r.FormValue("system")
	if top.SIselected == "" {
		top.SIselected = "Go"
	}
	top.Qa.System = systemlist[top.SIselected]
	question := r.FormValue("question")
	top.Qa.Modelname = r.FormValue("modelname")
	if top.Qa.Modelname == "" {
		top.Qa.Modelname = "deepseek-chat"
	}
	dsmt := r.FormValue("maxtokens")
	top.Qa.Maxtokens, _ = strconv.Atoi(dsmt)
	if top.Qa.Maxtokens == 0 {
		top.Qa.Maxtokens = 2000
	}
	top.Stmp = r.FormValue("temperature")
	if top.Stmp == "" {
		top.Stmp = "0.2"
	}
	top.Qa.Temperature, _ = strconv.ParseFloat(top.Stmp, 64)

	top.Target = r.FormValue("target")

	for i := 9; i >= 0; i-- {
		it := r.FormValue(fmt.Sprintf("checkbox%d", i))
		if it == "on" {
			history = append(history,
				qah{Sid: r.FormValue(fmt.Sprintf("id%d", i)),
					Question: r.FormValue(fmt.Sprintf("question%d", i)),
					Answer:   r.FormValue(fmt.Sprintf("answer%d", i)),
					Model:    r.FormValue(fmt.Sprintf("modelname%d", i))})
		}

	}

	if question != "" {
		top.Qa.Question = question
		apiKey := venderlist[modellist[top.Qa.Modelname].Vendor].Apikey
		url := venderlist[modellist[top.Qa.Modelname].Vendor].Url
		model := modellist[top.Qa.Modelname].Model
		switch model {
		case "deepseek":
			err = askOpenAI(&top.Qa, history, url, apiKey)
		case "claude":
			err = askClaude(&top.Qa, history, url, apiKey)
		case "gemini":
			err = askGemini(&top.Qa, history, url, apiKey)
		case "openai":
			err = askOpenAI(&top.Qa, history, url, apiKey)
		case "openai2":
			err = askOpenAI2(&top.Qa, history, url, apiKey)
		default:
			err = fmt.Errorf("invalid modelname: %s", top.Qa.Modelname)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}
		if err != nil {
			err = fmt.Errorf("API error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}

		qbu := top.Qa.Question
		top.Qa.Question += "\nwith ID:"
		for i := 0; i < len(history); i++ {
			if i != 0 {
				top.Qa.Question += ", "
			}
			top.Qa.Question += history[i].Sid
		}

		err = srdblib.Dbmap.Insert(&top.Qa)
		top.Qa.Question = qbu
		if err != nil {
			err = fmt.Errorf("Insert() Database error. err = %w", err)
			log.Printf("%s\n", err.Error())
			w.Write([]byte(err.Error()))
			return
		}
	}

	offset := (page - 1) * pageSize
	var intf []interface{}
	sqlst := ""
	if top.Target == "" {
		sqlst = "SELECT * FROM qa_records ORDER BY id DESC LIMIT ? OFFSET ? "
		intf, err = srdblib.Dbmap.Select(&Qa_recordsDB{}, sqlst, pageSize, offset)
	} else {
		sqlst = "SELECT * FROM qa_records "
		sqlst += " WHERE MATCH(question, answer) AGAINST( ? IN BOOLEAN MODE) "
		sqlst += " ORDER BY id DESC LIMIT ? OFFSET ? "
		intf, err = srdblib.Dbmap.Select(&Qa_recordsDB{}, sqlst, top.Target, pageSize, offset)
	}
	if err != nil {
		err = fmt.Errorf("Select() Database error. err = %w", err)
		log.Printf("%s\n", err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	top.Qalist = make([]Qa_recordsDB, 0)
	for _, v := range intf {
		top.Qalist = append(top.Qalist, *v.(*Qa_recordsDB))
	}

	if page > 1 {
		top.HasPrevious = true
		top.PreviousPage = page - 1
	}

	var totalRecords int64
	if top.Target == "" {
		err = srdblib.Dbmap.SelectOne(&totalRecords, "SELECT COUNT(*) FROM qa_records")
	} else {
		err = srdblib.Dbmap.SelectOne(&totalRecords,
			"SELECT COUNT(*) FROM qa_records WHERE MATCH(question, answer) AGAINST( ? IN BOOLEAN MODE)",
			top.Target)
	}
	if err != nil {
		err = fmt.Errorf("SelectOne() Database error. err = %w", err)
		log.Printf("%s\n", err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	if offset+pageSize < int(totalRecords) {
		top.HasNext = true
		top.NextPage = page + 1
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
			return template.CSS("hsl(270, 100%, 50%)")
		},
	}

	tpl := template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/dschat.html"))

	if err := tpl.ExecuteTemplate(w, "dschat.html", top); err != nil {
		err = fmt.Errorf("tpl.ExceuteTemplate(w,\"dschat.html\", top) err=%s", err.Error())
		log.Printf("err=%s\n", err.Error())
		w.Write([]byte(err.Error()))
	}
}

/*
func main() {
	http.HandleFunc("/dschat", ValidateJWT(HandlerDschat))
	http.HandleFunc("/signup", SignupHandler)
	http.HandleFunc("/verify", VerifyCodeHandler)
	http.HandleFunc("/login", LoginHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
*/
