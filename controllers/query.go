package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"rag-document-assistant/services"
)

type QueryController struct{ service *services.QueryService }

func NewQueryController(service *services.QueryService) *QueryController {
	return &QueryController{service: service}
}

type queryRequest struct {
	Question string `json:"question" binding:"required"`
	Debug    bool   `json:"debug"`
}

func (h *QueryController) Query(c *gin.Context) {
	var request queryRequest
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Question) == "" {
		writeError(c, http.StatusBadRequest, "invalid_question", "question is required")
		return
	}
	response, err := h.service.Answer(c.Request.Context(), request.Question, request.Debug)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *QueryController) Retrieve(c *gin.Context) {
	var request queryRequest
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Question) == "" {
		writeError(c, http.StatusBadRequest, "invalid_question", "question is required")
		return
	}
	vector, matches, err := h.service.Retrieve(c.Request.Context(), request.Question)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"question":  request.Question,
		"embedding": gin.H{"dimensions": len(vector)},
		"matches":   matches,
	})
}

func (h *QueryController) writeServiceError(c *gin.Context, err error) {
	slog.Error("query processing failed",
		"request_id", c.GetString("request_id"),
		"path", c.Request.URL.Path,
		"error", err,
	)
	if errors.Is(err, services.ErrInvalidInput) {
		writeError(c, http.StatusBadRequest, "invalid_question", err.Error())
		return
	}
	if strings.Contains(err.Error(), "API key is not configured") {
		writeError(c, http.StatusServiceUnavailable, "openai_not_configured", "OpenAI API key is not configured")
		return
	}
	writeError(c, http.StatusInternalServerError, "query_failed", "query processing failed")
}
