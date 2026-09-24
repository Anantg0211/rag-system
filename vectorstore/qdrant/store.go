package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"rag-document-assistant/models"
)

type Store struct {
	baseURL    string
	apiKey     string
	collection string
	dimensions int
	http       *http.Client
}

func New(baseURL, apiKey, collection string, dimensions int, client *http.Client) *Store {
	return &Store{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey,
		collection: collection, dimensions: dimensions, http: client}
}

func (s *Store) EnsureCollection(ctx context.Context) error {
	var current collectionResponse
	status, err := s.request(ctx, http.MethodGet, s.collectionPath(), nil, &current)
	if err != nil && status != http.StatusNotFound {
		return fmt.Errorf("inspect Qdrant collection: %w", err)
	}
	if status == http.StatusNotFound {
		_, err = s.request(ctx, http.MethodPut, s.collectionPath(), map[string]any{
			"vectors": map[string]any{"size": s.dimensions, "distance": "Cosine"},
		}, &map[string]any{})
		if err != nil {
			return fmt.Errorf("create Qdrant collection: %w", err)
		}
		return nil
	}
	if current.Result.Config.Params.Vectors.Size != s.dimensions {
		return fmt.Errorf("Qdrant collection %q uses %d dimensions; configuration requires %d",
			s.collection, current.Result.Config.Params.Vectors.Size, s.dimensions)
	}
	if !strings.EqualFold(current.Result.Config.Params.Vectors.Distance, "Cosine") {
		return fmt.Errorf("Qdrant collection %q uses %s distance; Cosine is required",
			s.collection, current.Result.Config.Params.Vectors.Distance)
	}
	return nil
}

func (s *Store) Upsert(ctx context.Context, chunks []models.Chunk, vectors [][]float32) error {
	if len(chunks) != len(vectors) {
		return fmt.Errorf("chunk/vector count mismatch: %d/%d", len(chunks), len(vectors))
	}
	points := make([]point, len(chunks))
	for i, chunk := range chunks {
		if len(vectors[i]) != s.dimensions {
			return fmt.Errorf("vector %d has %d dimensions, want %d", i, len(vectors[i]), s.dimensions)
		}
		points[i] = point{ID: chunk.ID, Vector: vectors[i], Payload: payload{
			DocumentID: chunk.DocumentID, Filename: chunk.Filename, Page: chunk.Page,
			ChunkIndex: chunk.ChunkIndex, Text: chunk.Text,
		}}
	}
	path := s.collectionPath() + "/points?wait=true"
	_, err := s.request(ctx, http.MethodPut, path, map[string]any{"points": points}, &map[string]any{})
	if err != nil {
		return fmt.Errorf("upsert Qdrant points: %w", err)
	}
	return nil
}

func (s *Store) Search(ctx context.Context, vector []float32, limit int, minimumScore float64) ([]models.SearchResult, error) {
	if len(vector) != s.dimensions {
		return nil, fmt.Errorf("query vector has %d dimensions, want %d", len(vector), s.dimensions)
	}
	var response queryResponse
	_, err := s.request(ctx, http.MethodPost, s.collectionPath()+"/points/query", map[string]any{
		"query": vector, "limit": limit, "score_threshold": minimumScore,
		"with_payload": true, "with_vector": false,
	}, &response)
	if err != nil {
		return nil, err
	}
	results := make([]models.SearchResult, 0, len(response.Result.Points))
	for _, item := range response.Result.Points {
		results = append(results, models.SearchResult{
			ChunkID: item.ID.String(), DocumentID: item.Payload.DocumentID,
			Document: item.Payload.Filename, Page: item.Payload.Page,
			ChunkIndex: item.Payload.ChunkIndex, Text: item.Payload.Text, Score: item.Score,
		})
	}
	return results, nil
}

func (s *Store) Ping(ctx context.Context) error {
	status, err := s.request(ctx, http.MethodGet, "/healthz", nil, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("Qdrant health returned %d", status)
	}
	return nil
}

func (s *Store) collectionPath() string {
	return "/collections/" + url.PathEscape(s.collection)
}

func (s *Store) request(ctx context.Context, method, path string, body, response any) (int, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("Qdrant returned %d: %s", resp.StatusCode, truncate(string(data), 500))
	}
	if response != nil && len(data) > 0 {
		if err := json.Unmarshal(data, response); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

type collectionResponse struct {
	Result struct {
		Config struct {
			Params struct {
				Vectors struct {
					Size     int    `json:"size"`
					Distance string `json:"distance"`
				} `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

type point struct {
	ID      string    `json:"id"`
	Vector  []float32 `json:"vector"`
	Payload payload   `json:"payload"`
}

type payload struct {
	DocumentID string `json:"document_id"`
	Filename   string `json:"filename"`
	Page       int    `json:"page"`
	ChunkIndex int    `json:"chunk_index"`
	Text       string `json:"text"`
}

type pointID json.RawMessage

func (id *pointID) UnmarshalJSON(data []byte) error {
	*id = append((*id)[:0], data...)
	return nil
}

func (id pointID) String() string {
	var value string
	if err := json.Unmarshal(id, &value); err == nil {
		return value
	}
	return strings.Trim(string(id), "\"")
}

type queryResponse struct {
	Result struct {
		Points []struct {
			ID      pointID `json:"id"`
			Score   float64 `json:"score"`
			Payload payload `json:"payload"`
		} `json:"points"`
	} `json:"result"`
}

func truncate(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
