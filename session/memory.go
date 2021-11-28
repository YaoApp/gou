package session

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/client"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
)

// Memory 内存
type Memory struct{}

// DMap DMap 数据接口
type DMap interface {
	Get(key string) (interface{}, error)
	Put(key string, value interface{}) error
	PutEx(key string, value interface{}, timeout time.Duration) error
	Query(q query.M) (Cursor, error)
}

// Cursor Cursor Interrface
type Cursor interface {
	Range(f func(key string, value interface{}) bool) error
}

// ServerDMap 服务端 DMap
type ServerDMap struct{ *olric.DMap }

// ClientDMap 客户端 DMap
type ClientDMap struct{ *client.DMap }

// Query 查询接口
func (dmap ServerDMap) Query(q query.M) (Cursor, error) {
	return dmap.DMap.Query(q)
}

// Query 查询接口
func (dmap ClientDMap) Query(q query.M) (Cursor, error) {
	return dmap.DMap.Query(q)
}

var dmap DMap

// MemoryUse 设置数据源
func MemoryUse(dm DMap) {
	dmap = dm
}

// Init 初始化
func (mem *Memory) Init() {}

// MemoryLocalServer 启动服务
func MemoryLocalServer() {
	c := config.New("local")
	c.Logger.SetOutput(ioutil.Discard) // 暂时关闭日志
	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		// log.Println("[INFO] Olric is ready to accept connections")
	}

	db, err := olric.New(c)
	if err != nil {
		log.Fatalf("Failed to create Olric instance: %v", err)
	}

	go func() {
		err = db.Start() // Call Start at background. It's a blocker call.
		if err != nil {
			log.Fatalf("olric.Start returned an error: %v", err)
		}
	}()

	<-ctx.Done()
	dm, err := db.NewDMap("local-session")
	if err != nil {
		log.Fatalf("olric.NewDMap returned an error: %v", err)
	}

	MemoryUse(ServerDMap{DMap: dm})
}

// Set 设置数值
func (mem *Memory) Set(id string, key string, value interface{}, timeout time.Duration) error {
	ckey := fmt.Sprintf("%s:%s", id, key)
	if timeout == 0 {
		return dmap.Put(ckey, value)
	}
	return dmap.PutEx(ckey, value, timeout)
}

// Get 读取数值
func (mem *Memory) Get(id string, key string) (interface{}, error) {
	ckey := fmt.Sprintf("%s:%s", id, key)
	value, err := dmap.Get(ckey)
	if err == olric.ErrKeyNotFound {
		return nil, nil
	}
	return value, err
}

// Dump 导出所有数据
func (mem *Memory) Dump(id string) (map[string]interface{}, error) {
	prefix := fmt.Sprintf("%s:", id)
	c, err := dmap.Query(query.M{"$onKey": query.M{"$regexMatch": "^" + prefix}})
	if err != nil {
		return nil, err
	}
	res := map[string]interface{}{}
	err = c.Range(func(key string, value interface{}) bool {
		key = strings.TrimPrefix(key, prefix)
		res[key] = value
		return true
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
