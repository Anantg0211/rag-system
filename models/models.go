package models

import "time"

const (
	DocumentStatusProcessing = "processing"
	DocumentStatusReady      = "ready"
	DocumentStatusFailed     = "failed"
)

type Document struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	PageCount    int       `json:"page_count"`
	ChunkCount   int       `json:"chunk_count"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PageText struct {
	Page int
	Text string
}

type Chunk struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Filename   string    `json:"filename"`
	Page       int       `json:"page"`
	ChunkIndex int       `json:"chunk_index"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

type SearchResult struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Document   string  `json:"document"`
	Page       int     `json:"page"`
	ChunkIndex int     `json:"chunk_index"`
	Text       string  `json:"text,omitempty"`
	Score      float64 `json:"score"`
}

type Source struct {
	Document   string  `json:"document"`
	Page       int     `json:"page"`
	ChunkID    string  `json:"chunk_id"`
	ChunkIndex int     `json:"chunk_index"`
	Score      float64 `json:"score"`
}

type EmbeddingDebug struct {
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
	Preview    []float32 `json:"preview"`
}

type QueryDebug struct {
	Stages            []string       `json:"stages"`
	QuestionEmbedding EmbeddingDebug `json:"question_embedding"`
	RetrievedChunks   []SearchResult `json:"retrieved_chunks"`
	ContextSentToLLM  string         `json:"context_sent_to_llm"`
}

type QueryResponse struct {
	Answer  string      `json:"answer"`
	Sources []Source    `json:"sources"`
	Debug   *QueryDebug `json:"debug,omitempty"`
}
