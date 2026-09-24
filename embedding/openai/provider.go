package openai

import (
	"context"
	"fmt"
	"sort"

	"rag-document-assistant/helper"
)

type Provider struct {
	client     *helper.OpenAIClient
	model      string
	dimensions int
	batchSize  int
}

func New(client *helper.OpenAIClient, model string, dimensions, batchSize int) *Provider {
	return &Provider{client: client, model: model, dimensions: dimensions, batchSize: batchSize}
}

func (p *Provider) Model() string { return p.model }

func (p *Provider) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}
	result := make([][]float32, 0, len(inputs))
	for start := 0; start < len(inputs); start += p.batchSize {
		end := start + p.batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		var response embeddingResponse
		if err := p.client.Post(ctx, "/embeddings", embeddingRequest{
			Input: inputs[start:end], Model: p.model, Dimensions: p.dimensions,
		}, &response); err != nil {
			return nil, err
		}
		sort.Slice(response.Data, func(i, j int) bool { return response.Data[i].Index < response.Data[j].Index })
		if len(response.Data) != end-start {
			return nil, fmt.Errorf("OpenAI returned %d embeddings for %d inputs", len(response.Data), end-start)
		}
		for _, item := range response.Data {
			if len(item.Embedding) != p.dimensions {
				return nil, fmt.Errorf("OpenAI embedding has %d dimensions, want %d", len(item.Embedding), p.dimensions)
			}
			vector := make([]float32, len(item.Embedding))
			for i, value := range item.Embedding {
				vector[i] = float32(value)
			}
			result = append(result, vector)
		}
	}
	return result, nil
}

type embeddingRequest struct {
	Input      []string `json:"input"`
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}
