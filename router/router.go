package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"rag-document-assistant/controllers"
	appmiddleware "rag-document-assistant/middleware"
)

func New(logger *slog.Logger, health *controllers.HealthController,
	documents *controllers.DocumentController, query *controllers.QueryController) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(appmiddleware.RequestLogger(logger), appmiddleware.Recovery(logger))
	router.GET("/health", health.Check)
	api := router.Group("/api")
	api.POST("/documents", documents.Upload)
	api.GET("/documents", documents.List)
	api.POST("/retrieve", query.Retrieve)
	api.POST("/query", query.Query)
	return router
}
