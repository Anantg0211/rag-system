package services

import (
	"context"
	"io"

	"rag-document-assistant/models"
)

type DocumentRepository interface {
	CreateDocument(context.Context, models.Document) error
	SaveChunks(context.Context, string, int, []models.Chunk) error
	MarkDocumentReady(context.Context, string) error
	MarkDocumentFailed(context.Context, string, string) error
	ListDocuments(context.Context) ([]models.Document, error)
}

type PDFExtractor interface {
	Extract(context.Context, io.Reader) ([]models.PageText, error)
}

type Chunker interface {
	Chunk(string, []models.PageText) []models.Chunk
}

type EmbeddingProvider interface {
	Embed(context.Context, []string) ([][]float32, error)
	Model() string
}

type VectorStore interface {
	Upsert(context.Context, []models.Chunk, [][]float32) error
	Search(context.Context, []float32, int, float64) ([]models.SearchResult, error)
	Ping(context.Context) error
}

type AnswerGenerator interface {
	Generate(context.Context, string, string) (string, error)
}

type HealthChecker interface{ Ping(context.Context) error }
