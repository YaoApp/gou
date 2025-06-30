package graphrag

import (
	"fmt"
	"os"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/graph/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/vector/qdrant"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
)

// ==== Test Utils ====

func GetTestConfigs() map[string]*Config {
	vectorStore := qdrant.NewStore()
	graphStore := neo4j.NewStore()
	logger := log.StandardLogger()

	return map[string]*Config{
		"vector":              {Vector: vectorStore},
		"vector+graph":        {Graph: graphStore, Vector: vectorStore},
		"vector+graph+logger": {Graph: graphStore, Vector: vectorStore, Logger: logger},
		"invalid":             {},
	}
}

// getVectorStore returns the vector store for the given name
func getVectorStore(name string) types.VectorStoreConfig {
	return types.VectorStoreConfig{
		Dimension:      128,
		Distance:       types.DistanceCosine,
		IndexType:      types.IndexTypeHNSW,
		CollectionName: fmt.Sprintf("test_connection_%s", name),
		ExtraParams: map[string]interface{}{
			"host": getEnvOrDefault("QDRANT_TEST_HOST", "localhost"),
			"port": getEnvOrDefault("QDRANT_TEST_PORT", "6334"),
		},
	}
}

func getGraphStore(name string) types.GraphStoreConfig {
	return types.GraphStoreConfig{
		StoreType:   "neo4j",
		DatabaseURL: getEnvOrDefault("NEO4J_TEST_URL", "neo4j://localhost:7687"),
		DriverConfig: map[string]interface{}{
			"username": getEnvOrDefault("NEO4J_TEST_USER", "neo4j"),
			"password": getEnvOrDefault("NEO4J_TEST_PASS", "Yao2026Neo4j"),
		},
	}
}

func getStore(name string) store.Store {
	conn, err := getStoreConnector(name)
	if err != nil {
		panic(err)
	}

	s, err := store.New(conn, nil)
	if err != nil {
		panic(err)
	}
	return s
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getOpenAIConnector(name, model string) (connector.Connector, error) {
	return connector.New("openai", fmt.Sprintf("test_%s", name), []byte(
		`{
			"label": "`+name+`",
			"options": {
				"key": "`+getEnvOrDefault("OPENAI_TEST_KEY", "")+`",
				"model": "`+model+`",
			}
		}`,
	))
}

func getDeepSeekConnector(name string) (connector.Connector, error) {
	model := getEnvOrDefault("RAG_LLM_TEST_SMODEL", "")
	return connector.New("openai", fmt.Sprintf("test_%s", name), []byte(
		`{
			"label": "`+name+`",
			"options": {
				"key": "`+getEnvOrDefault("RAG_LLM_TEST_KEY", "")+`",
				"model": "`+model+`",
				"proxy": "`+getEnvOrDefault("RAG_LLM_TEST_URL", "")+`"
			}
		}`,
	))
}

func getStoreConnector(name string) (connector.Connector, error) {
	return connector.New("redis", fmt.Sprintf("test_%s", name), []byte(
		`
		"label": "`+name+`", 
		"options": {
			{
				"host": "`+getEnvOrDefault("REDIS_TEST_HOST", "localhost")+`",
				"port": "`+getEnvOrDefault("REDIS_TEST_PORT", "6379")+`",
				"pass": "`+getEnvOrDefault("REDIS_TEST_PASS", "")+`",
				"db": "`+getEnvOrDefault("REDIS_TEST_DB", "5")+`"
			}
		}`,
	))
}
