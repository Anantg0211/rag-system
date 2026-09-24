package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"rag-document-assistant/models"
)

type DocumentService struct {
	repository DocumentRepository
	extractor  PDFExtractor
	chunker    Chunker
	embedder   EmbeddingProvider
	vectors    VectorStore
	logger     *slog.Logger
}

func NewDocumentService(repository DocumentRepository, extractor PDFExtractor, chunker Chunker,
	embedder EmbeddingProvider, vectors VectorStore, logger *slog.Logger) *DocumentService {
	return &DocumentService{repository: repository, extractor: extractor, chunker: chunker,
		embedder: embedder, vectors: vectors, logger: logger}
}

func (s *DocumentService) Upload(ctx context.Context, filename, contentType string, size int64, reader io.Reader) (models.Document, error) {
	document := models.Document{
		ID: uuid.NewString(), Filename: filename, ContentType: contentType,
		SizeBytes: size, Status: models.DocumentStatusProcessing,
	}
	if err := s.repository.CreateDocument(ctx, document); err != nil {
		return models.Document{}, err
	}

	fail := func(cause error) (models.Document, error) {
		message := safeFailure(cause)
		if markErr := s.repository.MarkDocumentFailed(context.WithoutCancel(ctx), document.ID, message); markErr != nil {
			s.logger.Error("failed to record document failure", "document_id", document.ID, "error", markErr)
		}
		document.Status = models.DocumentStatusFailed
		document.ErrorMessage = &message
		return document, cause
	}

	pages, err := s.extractor.Extract(ctx, reader)
	if err != nil {
		return fail(fmt.Errorf("extract PDF: %w", err))
	}
	chunks := s.chunker.Chunk(document.ID, pages)
	for i := range chunks {
		chunks[i].Filename = filename
	}
	if len(chunks) == 0 {
		return fail(ErrNoText)
	}
	if err := s.repository.SaveChunks(ctx, document.ID, len(pages), chunks); err != nil {
		return fail(err)
	}

	texts := make([]string, len(chunks))
	for i := range chunks {
		texts[i] = chunks[i].Text
	}
	vectors, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return fail(fmt.Errorf("embed chunks: %w", err))
	}
	if err := s.vectors.Upsert(ctx, chunks, vectors); err != nil {
		return fail(fmt.Errorf("index chunks: %w", err))
	}
	if err := s.repository.MarkDocumentReady(ctx, document.ID); err != nil {
		return fail(err)
	}

	document.PageCount = len(pages)
	document.ChunkCount = len(chunks)
	document.Status = models.DocumentStatusReady
	s.logger.Info("document indexed", "document_id", document.ID, "filename", filename,
		"pages", len(pages), "chunks", len(chunks))
	return document, nil
}

func (s *DocumentService) List(ctx context.Context) ([]models.Document, error) {
	return s.repository.ListDocuments(ctx)
}

func safeFailure(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}
