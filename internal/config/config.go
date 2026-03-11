package config

import (
	"os"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from env vars or .env file.
type Config struct {
	RepoPath     string
	DatabaseDSN  string
	QdrantURL    string
	Neo4jURL     string
	Neo4jUser    string
	Neo4jPass    string
	GeminiAPIKey string
	HTTPAddr     string
}

// Load reads configuration from environment variables and optional .env file.
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("DATABASE_DSN", "postgres://goatlas:goatlas@localhost:5432/goatlas")
	viper.SetDefault("QDRANT_URL", "http://localhost:6334")
	viper.SetDefault("NEO4J_URL", "bolt://localhost:7687")
	viper.SetDefault("NEO4J_USER", "neo4j")
	viper.SetDefault("NEO4J_PASS", "goatlas_neo4j")
	viper.SetDefault("HTTP_ADDR", ":8080")

	_ = viper.ReadInConfig() // ignore missing .env file

	repoPath := viper.GetString("REPO_PATH")
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}

	return &Config{
		RepoPath:     repoPath,
		DatabaseDSN:  viper.GetString("DATABASE_DSN"),
		QdrantURL:    viper.GetString("QDRANT_URL"),
		Neo4jURL:     viper.GetString("NEO4J_URL"),
		Neo4jUser:    viper.GetString("NEO4J_USER"),
		Neo4jPass:    viper.GetString("NEO4J_PASS"),
		GeminiAPIKey: viper.GetString("GEMINI_API_KEY"),
		HTTPAddr:     viper.GetString("HTTP_ADDR"),
	}, nil
}
