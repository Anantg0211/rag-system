package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"rag-document-assistant/models"
)

type DocumentRepository struct{ pool *pgxpool.Pool }

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

func (r *DocumentRepository) CreateDocument(ctx context.Context, document models.Document) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO documents (id, filename, content_type, size_bytes, status)
		VALUES ($1, $2, $3, $4, $5)`,
		document.ID, document.Filename, document.ContentType, document.SizeBytes, document.Status,
	)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

func (r *DocumentRepository) SaveChunks(ctx context.Context, documentID string, pageCount int, chunks []models.Chunk) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save chunks: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, chunk := range chunks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_chunks (id, document_id, page_number, chunk_index, text)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				page_number = EXCLUDED.page_number,
				chunk_index = EXCLUDED.chunk_index,
				text = EXCLUDED.text`,
			chunk.ID, documentID, chunk.Page, chunk.ChunkIndex, chunk.Text,
		); err != nil {
			return fmt.Errorf("insert chunk %s: %w", chunk.ID, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE documents SET page_count = $2, updated_at = NOW() WHERE id = $1`,
		documentID, pageCount,
	); err != nil {
		return fmt.Errorf("update document page count: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save chunks: %w", err)
	}
	return nil
}

func (r *DocumentRepository) MarkDocumentReady(ctx context.Context, documentID string) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE documents
		SET status = $2, error_message = NULL, updated_at = NOW()
		WHERE id = $1`, documentID, models.DocumentStatusReady)
	if err != nil {
		return fmt.Errorf("mark document ready: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("mark document ready: document %s not found", documentID)
	}
	return nil
}

func (r *DocumentRepository) MarkDocumentFailed(ctx context.Context, documentID, message string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE documents
		SET status = $2, error_message = $3, updated_at = NOW()
		WHERE id = $1`, documentID, models.DocumentStatusFailed, message)
	if err != nil {
		return fmt.Errorf("mark document failed: %w", err)
	}
	return nil
}

func (r *DocumentRepository) ListDocuments(ctx context.Context) ([]models.Document, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id::text, d.filename, d.content_type, d.size_bytes, d.page_count,
		       COUNT(c.id)::int, d.status, d.error_message, d.created_at, d.updated_at
		FROM documents d
		LEFT JOIN document_chunks c ON c.document_id = d.id
		GROUP BY d.id
		ORDER BY d.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	documents := make([]models.Document, 0)
	for rows.Next() {
		var document models.Document
		if err := rows.Scan(&document.ID, &document.Filename, &document.ContentType,
			&document.SizeBytes, &document.PageCount, &document.ChunkCount, &document.Status,
			&document.ErrorMessage, &document.CreatedAt, &document.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return documents, nil
}

func (r *DocumentRepository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }
