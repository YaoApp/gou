package connector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector/database"
	"github.com/yaoapp/gou/connector/fastembed"
	mongo "github.com/yaoapp/gou/connector/mongo"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/connector/redis"
)

func TestLoadMysql(t *testing.T) {
	file := prepare(t, "mysql")
	_, err := Load(file, "mysql")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["mysql"]
	if !has {
		t.Fatal("the mysql connector does not exist")
	}

	if !Connectors["mysql"].Is(DATABASE) {
		t.Fatal("the connector is not a DATABASE")
	}

	if _, ok := Connectors["mysql"].(*database.Xun); !ok {
		t.Fatal("the mysql connector is not a *database.Xun")
	}
}

func TestLoadSQLite(t *testing.T) {
	file := prepare(t, "sqlite")
	_, err := Load(file, "sqlite")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["sqlite"]
	if !has {
		t.Fatal("the sqlite connector does not exist")
	}

	if !Connectors["sqlite"].Is(DATABASE) {
		t.Fatal("the sqlite connector is not a DATABASE")
	}

	if _, ok := Connectors["sqlite"].(*database.Xun); !ok {
		t.Fatal("the connector is not a *database.Xun")
	}
	assert.Equal(t, "sqlite", Connectors["sqlite"].ID())
}

func TestLoadRedis(t *testing.T) {
	file := prepare(t, "redis")
	_, err := Load(file, "redis")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["redis"]
	if !has {
		t.Fatal("the redis connector does not exist")
	}

	if !Connectors["redis"].Is(REDIS) {
		t.Fatal("the redis connector is not a REDIS")
	}

	if _, ok := Connectors["redis"].(*redis.Connector); !ok {
		t.Fatal("the connector is not a *redis.Connector")
	}

	assert.Equal(t, "redis", Connectors["redis"].ID())
}

func TestLoadMongoDB(t *testing.T) {
	file := prepare(t, "mongo")
	_, err := Load(file, "mongo")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["mongo"]
	if !has {
		t.Fatal("the mongo connector does not exist")
	}

	if !Connectors["mongo"].Is(MONGO) {
		t.Fatal("the redis connector is not a MONGO")
	}

	if _, ok := Connectors["mongo"].(*mongo.Connector); !ok {
		t.Fatal("the connector is not a *mongo.Connector")
	}
	assert.Equal(t, "mongo", Connectors["mongo"].ID())
}

func TestLoadOpenAI(t *testing.T) {
	file := prepare(t, "openai")
	_, err := Load(file, "openai")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["openai"]
	if !has {
		t.Fatal("the openai connector does not exist")
	}

	if !Connectors["openai"].Is(OPENAI) {
		t.Fatal("the connector is not a OPENAI")
	}

	if _, ok := Connectors["openai"].(*openai.Connector); !ok {
		t.Fatal("the openai connector is not a *openai.Connector")
	}

	setting := Connectors["openai"].Setting()
	assert.Equal(t, "openai", Connectors["openai"].ID())
	assert.Contains(t, setting["key"], "sk-")
}

func TestLoadFastembed(t *testing.T) {
	file := prepare(t, "fastembed")
	_, err := Load(file, "fastembed")
	if err != nil {
		t.Fatal(err)
	}

	_, has := Connectors["fastembed"]
	if !has {
		t.Fatal("the fastembed connector does not exist")
	}

	if !Connectors["fastembed"].Is(FASTEMBED) {
		t.Fatal("the connector is not a FASTEMBED")
	}

	if _, ok := Connectors["fastembed"].(*fastembed.Connector); !ok {
		t.Fatal("the fastembed connector is not a *fastembed.Connector")
	}

	setting := Connectors["fastembed"].Setting()
	assert.Equal(t, "fastembed", Connectors["fastembed"].ID())
	assert.NotEmpty(t, setting["host"])
	assert.NotEmpty(t, setting["model"])
}

func prepare(t *testing.T, name string) string {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
	return filepath.Join("connectors", fmt.Sprintf("%s.conn.yao", name))
}
