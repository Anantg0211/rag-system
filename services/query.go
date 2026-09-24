package services

import (
	"context"
	"fmt"
	"strings"

	"rag-document-assistant/models"
)

type QueryService struct {
	embedder        EmbeddingProvider
	vectors         VectorStore
	generator       AnswerGenerator
	topK            int
	minimumScore    float64
	maxContextChars int
}

func NewQueryService(embedder EmbeddingProvider, vectors VectorStore, generator AnswerGenerator,
	topK int, minimumScore float64, maxContextChars int) *QueryService {
	return &QueryService{embedder: embedder, vectors: vectors, generator: generator,
		topK: topK, minimumScore: minimumScore, maxContextChars: maxContextChars}
}

func (s *QueryService) Retrieve(ctx context.Context, question string) ([]float32, []models.SearchResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, nil, fmt.Errorf("%w: question is required", ErrInvalidInput)
	}
	vectors, err := s.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, nil, fmt.Errorf("embed question: %w", err)
	}
	if len(vectors) != 1 {
		return nil, nil, fmt.Errorf("embed question: expected one vector, got %d", len(vectors))
	}
	matches, err := s.vectors.Search(ctx, vectors[0], s.topK, s.minimumScore)
	if err != nil {
		return nil, nil, fmt.Errorf("search chunks: %w", err)
	}
	return vectors[0], matches, nil
}

func (s *QueryService) Answer(ctx context.Context, question string, debug bool) (models.QueryResponse, error) {
	vector, matches, err := s.Retrieve(ctx, question)
	if err != nil {
		return models.QueryResponse{}, err
	}

	contextText := buildContext(matches, s.maxContextChars)
	response := models.QueryResponse{Answer: InsufficientInformation, Sources: make([]models.Source, 0)}
	if len(matches) > 0 {
		answer, err := s.generator.Generate(ctx, strings.TrimSpace(question), contextText)
		if err != nil {
			return models.QueryResponse{}, fmt.Errorf("generate answer: %w", err)
		}
		answer = strings.TrimSpace(answer)
		if answer != "" {
			response.Answer = answer
		}
		for _, match := range matches {
			response.Sources = append(response.Sources, models.Source{
				Document: match.Document, Page: match.Page, ChunkID: match.ChunkID,
				ChunkIndex: match.ChunkIndex, Score: match.Score,
			})
		}
	}
	if debug {
		previewLength := 8
		if len(vector) < previewLength {
			previewLength = len(vector)
		}
		response.Debug = &models.QueryDebug{
			Stages:            []string{"question", "question_embedding", "qdrant_search", "retrieved_chunks", "context_to_llm", "final_answer"},
			QuestionEmbedding: models.EmbeddingDebug{Model: s.embedder.Model(), Dimensions: len(vector), Preview: vector[:previewLength]},
			RetrievedChunks:   matches, ContextSentToLLM: contextText,
		}
	}
	return response, nil
}

func buildContext(matches []models.SearchResult, maxChars int) string {
	var builder strings.Builder
	for i, match := range matches {
		block := fmt.Sprintf("[Source %d | document=%s | page=%d | chunk_id=%s | chunk_index=%d]\n%s\n\n",
			i+1, match.Document, match.Page, match.ChunkID, match.ChunkIndex, match.Text)
		if maxChars > 0 && builder.Len()+len(block) > maxChars {
			remaining := maxChars - builder.Len()
			if remaining > 0 {
				builder.WriteString(block[:remaining])
			}
			break
		}
		builder.WriteString(block)
	}
	return strings.TrimSpace(builder.String())
}
