package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/buntdb"
	"github.com/yaoapp/kun/log"
)

// BuntDB BuntDB store
type BuntDB struct {
	db *buntdb.DB
}

// NewBuntDB create a new BuntDB instance
func NewBuntDB(datafile string) (*BuntDB, error) {
	db := &BuntDB{db: nil}
	if _, err := os.Stat(filepath.Dir(datafile)); err == nil {
		data, err := buntdb.Open(datafile)
		if err != nil {
			return nil, err
		}
		db.db = data
		return db, nil
	}

	data, err := buntdb.Open(":memory:")
	if err != nil {
		return nil, err
	}
	db.db = data
	return db, nil
}

// Init initialization
func (bunt *BuntDB) Init() {}

// Set session value
func (bunt *BuntDB) Set(id string, key string, value interface{}, timeout time.Duration) error {
	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("Session buntdb Set: %s key %s", err.Error(), skey)
		return err
	}

	var option *buntdb.SetOptions = nil
	if timeout > 0 {
		option = &buntdb.SetOptions{Expires: true, TTL: timeout}
	}

	err = bunt.db.Update(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(skey, string(bytes), option)
		return err
	})

	if err != nil {
		log.Error("Session buntdb Set: %s key %s", err.Error(), skey)
		return err
	}

	return nil
}

// Get session value
func (bunt *BuntDB) Get(id string, key string) (interface{}, error) {
	return nil, nil
}

// Dump session data
func (bunt *BuntDB) Dump(id string) (map[string]interface{}, error) {
	return nil, nil
}
