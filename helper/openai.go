package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenAIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewOpenAIClient(baseURL, apiKey string, client *http.Client) *OpenAIClient {
	return &OpenAIClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: client}
}

func (c *OpenAIClient) Post(ctx context.Context, path string, requestBody, responseBody any) error {
	if strings.TrimSpace(c.apiKey) == "" {
		return fmt.Errorf("OpenAI API key is not configured")
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode OpenAI request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send OpenAI request: %w", err)
	}
	defer resp.Body.Close()
	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read OpenAI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OpenAI API returned %d: %s", resp.StatusCode, truncate(string(responseBytes), 500))
	}
	if err := json.Unmarshal(responseBytes, responseBody); err != nil {
		return fmt.Errorf("decode OpenAI response: %w", err)
	}
	return nil
}

func truncate(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
