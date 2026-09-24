package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rag-document-assistant/chunking"
	"rag-document-assistant/config"
	"rag-document-assistant/controllers"
	"rag-document-assistant/db"
	pdfextractor "rag-document-assistant/document/pdf"
	embeddingopenai "rag-document-assistant/embedding/openai"
	generationopenai "rag-document-assistant/generation/openai"
	"rag-document-assistant/helper"
	"rag-document-assistant/migrations"
	"rag-document-assistant/router"
	"rag-document-assistant/server"
	"rag-document-assistant/services"
	"rag-document-assistant/vectorstore/qdrant"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}
	cfg, err := config.Load(environment, "config")
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger := newLogger(cfg.Server.LogLevel)
	slog.SetDefault(logger)

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := db.Connect(startupCtx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.ApplyMigrations(startupCtx, pool, migrations.FS); err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	openAIClient := helper.NewOpenAIClient(cfg.OpenAI.BaseURL, cfg.OpenAI.APIKey, httpClient)
	embedder := embeddingopenai.New(openAIClient, cfg.OpenAI.EmbeddingModel,
		cfg.OpenAI.EmbeddingDimensions, cfg.OpenAI.EmbeddingBatchSize)
	generator := generationopenai.New(openAIClient, cfg.OpenAI.ChatModel)
	vectorStore := qdrant.New(cfg.Qdrant.URL, cfg.Qdrant.APIKey, cfg.Qdrant.Collection,
		cfg.OpenAI.EmbeddingDimensions, httpClient)
	if err := vectorStore.EnsureCollection(startupCtx); err != nil {
		return err
	}

	repository := db.NewDocumentRepository(pool)
	documentService := services.NewDocumentService(repository, pdfextractor.New(),
		chunking.New(cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap), embedder, vectorStore, logger)
	queryService := services.NewQueryService(embedder, vectorStore, generator,
		cfg.RAG.RetrievalTopK, cfg.RAG.RetrievalMinScore, cfg.RAG.MaxContextChars)

	healthController := controllers.NewHealthController(repository, vectorStore)
	documentController := controllers.NewDocumentController(documentService, cfg.Upload.MaxBytes)
	queryController := controllers.NewQueryController(queryService)
	handler := router.New(logger, healthController, documentController, queryController)
	httpServer := server.New(cfg.Server, handler, logger)

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Start() }()
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout())
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	switch strings.ToLower(level) {
	case "debug":
		parsed = slog.LevelDebug
	case "warn":
		parsed = slog.LevelWarn
	case "error":
		parsed = slog.LevelError
	default:
		parsed = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsed}))
}
