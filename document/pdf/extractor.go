package pdf

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	pdflib "github.com/ledongthuc/pdf"

	"rag-document-assistant/models"
)

type Extractor struct{}

func New() *Extractor { return &Extractor{} }

func (e *Extractor) Extract(ctx context.Context, reader io.Reader) ([]models.PageText, error) {
	temporary, err := os.CreateTemp("", "rag-upload-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temporary PDF: %w", err)
	}
	path := temporary.Name()
	defer os.Remove(path)

	if _, err := io.Copy(temporary, reader); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("copy PDF: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close temporary PDF: %w", err)
	}

	file, parsed, err := pdflib.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open PDF: %w", err)
	}
	defer file.Close()

	pages := make([]models.PageText, 0, parsed.NumPage())
	for pageNumber := 1; pageNumber <= parsed.NumPage(); pageNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page := parsed.Page(pageNumber)
		if page.V.IsNull() {
			pages = append(pages, models.PageText{Page: pageNumber})
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return nil, fmt.Errorf("extract page %d: %w", pageNumber, err)
		}
		pages = append(pages, models.PageText{Page: pageNumber, Text: strings.TrimSpace(text)})
	}
	return pages, nil
}
