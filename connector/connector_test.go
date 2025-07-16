package connector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sync"

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

func TestLoadSync(t *testing.T) {
	// Clean up existing connectors
	Connectors = map[string]Connector{}

	var wg sync.WaitGroup
	wg.Add(3)

	// Load multiple connectors concurrently
	go func() {
		defer wg.Done()
		_, err := LoadSync(prepare(t, "mysql"), "mysql-sync")
		assert.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		_, err := LoadSync(prepare(t, "redis"), "redis-sync")
		assert.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		_, err := LoadSync(prepare(t, "mongo"), "mongo-sync")
		assert.NoError(t, err)
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all connectors are loaded correctly
	connectors := []string{"mysql-sync", "redis-sync", "mongo-sync"}
	for _, id := range connectors {
		conn, has := Connectors[id]
		if !has {
			t.Fatalf("connector %s not loaded", id)
		}
		assert.NotNil(t, conn, "connector %s should not be nil", id)
	}

	// Verify connector types
	assert.True(t, Connectors["mysql-sync"].Is(DATABASE))
	assert.True(t, Connectors["redis-sync"].Is(REDIS))
	assert.True(t, Connectors["mongo-sync"].Is(MONGO))
}

func TestLoadSourceSync(t *testing.T) {
	// Clean up existing connectors
	Connectors = map[string]Connector{}

	// Prepare test data
	mysqlSource := []byte(`{
		"type": "mysql",
		"name": "MySQL Test",
		"version": "1.0.0",
		"options": {
			"database": "test",
			"host": "127.0.0.1",
			"port": "3306",
			"user": "root",
			"password": "123456"
		}
	}`)

	// redisSource := []byte(`{
	// 	"type": "redis",
	// 	"name": "Redis Test",
	// 	"version": "1.0.0",
	// 	"options": {
	// 		"host": "127.0.0.1",
	// 		"port": "6379",
	// 		"db": "0",
	// 		"timeout": 5
	// 	}
	// }`)

	var wg sync.WaitGroup
	wg.Add(1)

	errs := []error{}
	var errMu sync.Mutex

	// Load connectors concurrently
	go func() {
		defer wg.Done()
		_, err := LoadSourceSync(mysqlSource, "mysql-source-sync", "mysql.source.conn.yao")
		if err != nil {
			errMu.Lock()
			errs = append(errs, fmt.Errorf("mysql: %v", err))
			errMu.Unlock()
		}
	}()

	// go func() {
	// 	defer wg.Done()
	// 	_, err := LoadSourceSync(redisSource, "redis-source-sync", "redis.source.conn.yao")
	// 	if err != nil {
	// 		errMu.Lock()
	// 		errs = append(errs, fmt.Errorf("redis: %v", err))
	// 		errMu.Unlock()
	// 	}
	// }()

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}

	// Verify connectors are loaded correctly
	mysql, has := Connectors["mysql-source-sync"]
	assert.True(t, has, "mysql connector should be loaded")
	assert.NotNil(t, mysql, "mysql connector should not be nil")
	assert.True(t, mysql.Is(DATABASE))

	// redis, has := Connectors["redis-source-sync"]
	// assert.True(t, has, "redis connector should be loaded")
	// assert.NotNil(t, redis, "redis connector should not be nil")
	// assert.True(t, redis.Is(REDIS))
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
