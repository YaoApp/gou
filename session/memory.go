package session

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
)

// Memory 内存
type Memory struct{}

var dmap *olric.DMap

// MemoryUse 设置数据源
func MemoryUse(dm *olric.DMap) {
	dmap = dm
}

// Init 初始化
func (mem *Memory) Init() {
	if dmap == nil {
		mem.Local()
	}
}

// Local 启动服务
func (mem *Memory) Local() {
	c := config.New("local")
	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		log.Println("[INFO] Olric is ready to accept connections")
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

	MemoryUse(dm)
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
