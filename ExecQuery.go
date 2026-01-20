package main

import (
	"context"
	"fmt"
	// "log"
	"net/http"

	"google.golang.org/genai"
)

func ExecQuery(
	apikey string,
	model string,
	systemInstruction *genai.Content,
	userQuery []*genai.Content,
	temperature float32,
	maxOutputTokens int,
) (
	result *genai.GenerateContentResponse,
	err error,
) {
	httpClient := http.DefaultClient
	ctx := context.Background()
	clientConfig := &genai.ClientConfig{
		APIKey:     apikey,
		HTTPClient: httpClient,
		Backend:    genai.BackendGeminiAPI,
		// location and API key are mutually exclusive in the client initializer
		// Location:   "asia-northeast1",
	}
	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		err = fmt.Errorf("failed to create genai client: %w", err)
		return
	}

	gcc := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Temperature:       &temperature,
		MaxOutputTokens:   int32(maxOutputTokens),
	}

	result, err = client.Models.GenerateContent(
		ctx,
		model,
		userQuery,
		gcc,
	)
	if err != nil {
		err = fmt.Errorf("failed to generate content: %w", err)
		return
	}
	fmt.Println(result.Text())
	return
}
