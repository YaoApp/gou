package connector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector/database"
	mongo "github.com/yaoapp/gou/connector/mongo"
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
		t.Fatal("the mysql connector is not a DATABASE")
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
		t.Fatal("the sqlite connector is not a *database.Xun")
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
		t.Fatal("the redis connector is not a *redis.Connector")
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
		t.Fatal("the mongo connector is not a *mongo.Connector")
	}

	assert.Equal(t, "mongo", Connectors["mongo"].ID())
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
