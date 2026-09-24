package controllers

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"rag-document-assistant/services"
)

type DocumentController struct {
	service  *services.DocumentService
	maxBytes int64
}

func NewDocumentController(service *services.DocumentService, maxBytes int64) *DocumentController {
	return &DocumentController{service: service, maxBytes: maxBytes}
}

func (h *DocumentController) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_upload", "multipart field 'file' containing a PDF is required")
		return
	}
	if header.Size <= 0 {
		writeError(c, http.StatusBadRequest, "empty_file", "uploaded PDF is empty")
		return
	}
	if header.Size > h.maxBytes {
		writeError(c, http.StatusRequestEntityTooLarge, "file_too_large", "uploaded PDF exceeds the configured size limit")
		return
	}
	if !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		writeError(c, http.StatusUnsupportedMediaType, "unsupported_file", "only PDF files are supported")
		return
	}
	file, err := header.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_upload", "could not open uploaded PDF")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/pdf"
	}
	document, err := h.service.Upload(c.Request.Context(), filepath.Base(header.Filename), contentType, header.Size, file)
	if err != nil {
		if errors.Is(err, services.ErrNoText) {
			writeError(c, http.StatusUnprocessableEntity, "no_extractable_text", services.ErrNoText.Error())
			return
		}
		if strings.Contains(err.Error(), "API key is not configured") {
			writeError(c, http.StatusServiceUnavailable, "openai_not_configured", "OpenAI API key is not configured")
			return
		}
		writeError(c, http.StatusInternalServerError, "ingestion_failed", "document ingestion failed")
		return
	}
	c.JSON(http.StatusCreated, document)
}

func (h *DocumentController) List(c *gin.Context) {
	documents, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "list_failed", "could not list documents")
		return
	}
	c.JSON(http.StatusOK, gin.H{"documents": documents})
}
