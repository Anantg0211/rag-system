package chunking

import (
	"strings"
	"unicode"

	"github.com/google/uuid"

	"rag-document-assistant/models"
)

type Chunker struct {
	size    int
	overlap int
}

func New(size, overlap int) *Chunker { return &Chunker{size: size, overlap: overlap} }

func (c *Chunker) Chunk(documentID string, pages []models.PageText) []models.Chunk {
	chunks := make([]models.Chunk, 0)
	chunkIndex := 0
	for _, page := range pages {
		text := normalizeWhitespace(page.Text)
		runes := []rune(text)
		for start := 0; start < len(runes); {
			end := start + c.size
			if end > len(runes) {
				end = len(runes)
			}
			if end < len(runes) {
				minimumBreak := start + c.size/2
				for candidate := end; candidate > minimumBreak; candidate-- {
					if unicode.IsSpace(runes[candidate-1]) {
						end = candidate
						break
					}
				}
			}
			chunkText := strings.TrimSpace(string(runes[start:end]))
			if chunkText != "" {
				chunks = append(chunks, models.Chunk{
					ID: uuid.NewString(), DocumentID: documentID, Page: page.Page,
					ChunkIndex: chunkIndex, Text: chunkText,
				})
				chunkIndex++
			}
			if end == len(runes) {
				break
			}
			next := end - c.overlap
			if next <= start {
				next = end
			}
			start = next
		}
	}
	return chunks
}

func normalizeWhitespace(value string) string { return strings.Join(strings.Fields(value), " ") }
