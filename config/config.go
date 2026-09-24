package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string         `yaml:"-"`
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Qdrant      QdrantConfig   `yaml:"qdrant"`
	OpenAI      OpenAIConfig   `yaml:"openai"`
	RAG         RAGConfig      `yaml:"rag"`
	Upload      UploadConfig   `yaml:"upload"`
}

type ServerConfig struct {
	Port                   int    `yaml:"port"`
	LogLevel               string `yaml:"log_level"`
	ReadTimeoutSeconds     int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds    int    `yaml:"write_timeout_seconds"`
	ShutdownTimeoutSeconds int    `yaml:"shutdown_timeout_seconds"`
}

func (c ServerConfig) Address() string { return fmt.Sprintf(":%d", c.Port) }
func (c ServerConfig) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSeconds) * time.Second
}
func (c ServerConfig) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutSeconds) * time.Second
}
func (c ServerConfig) ShutdownTimeout() time.Duration {
	return time.Duration(c.ShutdownTimeoutSeconds) * time.Second
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type QdrantConfig struct {
	URL        string `yaml:"url"`
	APIKey     string `yaml:"api_key"`
	Collection string `yaml:"collection"`
}

type OpenAIConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	EmbeddingModel      string `yaml:"embedding_model"`
	EmbeddingDimensions int    `yaml:"embedding_dimensions"`
	ChatModel           string `yaml:"chat_model"`
	EmbeddingBatchSize  int    `yaml:"embedding_batch_size"`
}

type RAGConfig struct {
	ChunkSize         int     `yaml:"chunk_size"`
	ChunkOverlap      int     `yaml:"chunk_overlap"`
	RetrievalTopK     int     `yaml:"retrieval_top_k"`
	RetrievalMinScore float64 `yaml:"retrieval_min_score"`
	MaxContextChars   int     `yaml:"max_context_chars"`
}

type UploadConfig struct {
	MaxBytes int64 `yaml:"max_bytes"`
}

func Load(environment, configDir string) (Config, error) {
	if environment == "" {
		environment = "development"
	}
	if configDir == "" {
		configDir = "config"
	}

	path := filepath.Join(configDir, environment+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.Environment = environment
	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnvironment(cfg *Config) error {
	stringValues := []struct {
		name   string
		target *string
	}{
		{"LOG_LEVEL", &cfg.Server.LogLevel}, {"DATABASE_URL", &cfg.Database.URL},
		{"QDRANT_URL", &cfg.Qdrant.URL}, {"QDRANT_API_KEY", &cfg.Qdrant.APIKey},
		{"QDRANT_COLLECTION", &cfg.Qdrant.Collection}, {"OPENAI_API_KEY", &cfg.OpenAI.APIKey},
		{"OPENAI_BASE_URL", &cfg.OpenAI.BaseURL}, {"OPENAI_EMBEDDING_MODEL", &cfg.OpenAI.EmbeddingModel},
		{"OPENAI_CHAT_MODEL", &cfg.OpenAI.ChatModel},
	}
	for _, item := range stringValues {
		if value, ok := os.LookupEnv(item.name); ok {
			*item.target = value
		}
	}

	intValues := []struct {
		name   string
		target *int
	}{
		{"HTTP_PORT", &cfg.Server.Port}, {"OPENAI_EMBEDDING_DIMENSIONS", &cfg.OpenAI.EmbeddingDimensions},
		{"OPENAI_EMBEDDING_BATCH_SIZE", &cfg.OpenAI.EmbeddingBatchSize}, {"CHUNK_SIZE", &cfg.RAG.ChunkSize},
		{"CHUNK_OVERLAP", &cfg.RAG.ChunkOverlap}, {"RETRIEVAL_TOP_K", &cfg.RAG.RetrievalTopK},
		{"MAX_CONTEXT_CHARS", &cfg.RAG.MaxContextChars},
	}
	for _, item := range intValues {
		if value, ok := os.LookupEnv(item.name); ok {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s must be an integer: %w", item.name, err)
			}
			*item.target = parsed
		}
	}
	if value, ok := os.LookupEnv("MAX_UPLOAD_BYTES"); ok {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("MAX_UPLOAD_BYTES must be an integer: %w", err)
		}
		cfg.Upload.MaxBytes = parsed
	}
	if value, ok := os.LookupEnv("RETRIEVAL_MIN_SCORE"); ok {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("RETRIEVAL_MIN_SCORE must be a number: %w", err)
		}
		cfg.RAG.RetrievalMinScore = parsed
	}
	return nil
}

func (c Config) Validate() error {
	var problems []string
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, "server port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		problems = append(problems, "database URL is required")
	}
	if strings.TrimSpace(c.Qdrant.URL) == "" {
		problems = append(problems, "Qdrant URL is required")
	}
	if strings.TrimSpace(c.Qdrant.Collection) == "" {
		problems = append(problems, "Qdrant collection is required")
	}
	if strings.TrimSpace(c.OpenAI.BaseURL) == "" {
		problems = append(problems, "OpenAI base URL is required")
	}
	if strings.TrimSpace(c.OpenAI.EmbeddingModel) == "" {
		problems = append(problems, "embedding model is required")
	}
	if c.OpenAI.EmbeddingDimensions < 1 {
		problems = append(problems, "embedding dimensions must be positive")
	}
	if c.OpenAI.EmbeddingBatchSize < 1 {
		problems = append(problems, "embedding batch size must be positive")
	}
	if c.RAG.ChunkSize < 1 {
		problems = append(problems, "chunk size must be positive")
	}
	if c.RAG.ChunkOverlap < 0 || c.RAG.ChunkOverlap >= c.RAG.ChunkSize {
		problems = append(problems, "chunk overlap must be non-negative and smaller than chunk size")
	}
	if c.RAG.RetrievalTopK < 1 {
		problems = append(problems, "retrieval Top-K must be positive")
	}
	if c.RAG.RetrievalMinScore < -1 || c.RAG.RetrievalMinScore > 1 {
		problems = append(problems, "retrieval minimum score must be between -1 and 1")
	}
	if c.Upload.MaxBytes < 1 {
		problems = append(problems, "maximum upload size must be positive")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
