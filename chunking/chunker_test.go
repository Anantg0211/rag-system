package chunking

import (
	"strings"
	"testing"

	"rag-document-assistant/models"
)

func TestChunkPreservesPageAndOverlap(t *testing.T) {
	chunker := New(20, 5)
	chunks := chunker.Chunk("document-1", []models.PageText{{Page: 3, Text: "alpha bravo charlie delta echo foxtrot"}})
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.Page != 3 {
			t.Errorf("chunk %d page = %d, want 3", i, chunk.Page)
		}
		if chunk.ChunkIndex != i {
			t.Errorf("chunk index = %d, want %d", chunk.ChunkIndex, i)
		}
		if len([]rune(chunk.Text)) > 20 {
			t.Errorf("chunk exceeds configured size: %q", chunk.Text)
		}
	}
	if !strings.Contains(chunks[0].Text+" "+chunks[1].Text, "charlie") {
		t.Fatalf("expected boundary content to remain present: %#v", chunks)
	}
}

func TestChunkNormalizesWhitespaceAndSkipsEmptyPages(t *testing.T) {
	chunker := New(100, 10)
	chunks := chunker.Chunk("document-1", []models.PageText{
		{Page: 1, Text: "  Mr   Anant\n has jaundice.  "}, {Page: 2, Text: "  \n\t "},
	})
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Text != "Mr Anant has jaundice." {
		t.Fatalf("unexpected text %q", chunks[0].Text)
	}
}
