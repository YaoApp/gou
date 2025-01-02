package rag

import (
	"fmt"
	"strconv"

	"github.com/yaoapp/gou/rag/driver"
	"github.com/yaoapp/gou/rag/driver/openai"
	"github.com/yaoapp/gou/rag/driver/qdrant"
)

const (
	// DriverQdrant is the Qdrant vector store driver
	DriverQdrant = "qdrant"
	// DriverOpenAI is the OpenAI embeddings driver
	DriverOpenAI = "openai"
)

// NewEngine creates a new RAG engine instance
func NewEngine(driverName string, config driver.IndexConfig, vectorizer driver.Vectorizer) (driver.Engine, error) {
	switch driverName {
	case DriverQdrant:
		// Convert IndexConfig to qdrant.Config
		qConfig := qdrant.Config{
			Host:       config.Options["host"],
			APIKey:     config.Options["api_key"],
			Vectorizer: vectorizer,
		}
		if portStr, ok := config.Options["port"]; ok {
			if port, err := strconv.ParseUint(portStr, 10, 32); err == nil {
				qConfig.Port = uint32(port)
			}
		}
		return qdrant.NewEngine(qConfig)
	default:
		return nil, fmt.Errorf("unsupported engine driver: %s", driverName)
	}
}

// NewVectorizer creates a new vectorizer instance
func NewVectorizer(driverName string, config driver.VectorizeConfig) (driver.Vectorizer, error) {
	switch driverName {
	case DriverOpenAI:
		return openai.New(openai.Config{
			APIKey: config.Options["api_key"],
			Model:  config.Model,
		})
	default:
		return nil, fmt.Errorf("unsupported vectorizer driver: %s", driverName)
	}
}

// NewFileUpload creates a new file upload instance
func NewFileUpload(driverName string, engine driver.Engine, vectorizer driver.Vectorizer) (driver.FileUpload, error) {
	switch driverName {
	case DriverQdrant:
		if qEngine, ok := engine.(*qdrant.Engine); ok {
			return qdrant.NewFileUpload(qEngine, vectorizer)
		}
		return nil, fmt.Errorf("engine type mismatch: expected *qdrant.Engine")
	default:
		return nil, fmt.Errorf("unsupported file upload driver: %s", driverName)
	}
}
