package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if _, err := os.Stat(filepath.Dir(datafile)); datafile != "" && datafile != ":memory:" && err == nil {
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

// Close burndb connection
func (bunt *BuntDB) Close() {
	bunt.db.Close()
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

	log.Debug("Session buntdb Set: %s KEY: %s VALUE: %v TS: %#v", skey, key, value, option.TTL)
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
	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	var value interface{} = nil
	err := bunt.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(skey)
		if err != nil {
			if err.Error() == "not found" {
				return nil
			}
			log.Error("Session buntdb Get: %s ERROR:%s", skey, err.Error())
			return err
		}

		err = jsoniter.Unmarshal([]byte(val), &value)
		if err != nil {
			log.Error("Session buntdb Get JSON: %s val: %s ERROR:%s", skey, val, err.Error())
			return err
		}
		return nil
	})

	return value, err
}

// Del session value
func (bunt *BuntDB) Del(id string, key string) error {
	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	err := bunt.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(skey)
		return err
	})

	if err != nil {
		log.Error("Session buntdb Del: %s key %s", err.Error(), skey)
		return err
	}
	return nil
}

// Dump session data
func (bunt *BuntDB) Dump(id string) (map[string]interface{}, error) {
	prefix := fmt.Sprintf("%s:%s:", "yao:session", id)
	res := map[string]interface{}{}
	err := bunt.db.View(func(tx *buntdb.Tx) error {
		err := tx.AscendKeys("yao:session:"+id+":*", func(key, val string) bool {
			var value interface{} = nil
			key = strings.TrimPrefix(key, prefix)
			err := jsoniter.Unmarshal([]byte(val), &value)
			if err != nil {
				log.Error("Session buntdb Get JSON: %s val: %s ERROR:%s", key, val, err.Error())
				return true
			}
			res[key] = value
			return true // continue iteration
		})

		if err != nil {
			if err.Error() == "not found" {
				return nil
			}
			log.Error("Session buntdb Dump %s ERROR:%s", id, err.Error())
			return err
		}
		return err
	})
	return res, err
}
