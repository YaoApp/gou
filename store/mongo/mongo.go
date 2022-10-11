package mongo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	mongodb "github.com/yaoapp/gou/connector/mongo"
	"github.com/yaoapp/kun/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// New create a new store via connector
func New(c connector.Connector) (*Store, error) {
	mongodb, ok := c.(*mongodb.Connector)
	if !ok {
		return nil, fmt.Errorf("the connector was not a *redis.Connector")
	}

	// coll
	coll := mongodb.Database.Collection(mongodb.ID())

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: -1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := coll.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return nil, err
	}

	return &Store{Database: mongodb.Database, Collection: coll}, nil
}

// Get looks up a key's value from the store.
func (store *Store) Get(key string) (value interface{}, ok bool) {
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: key}}).Decode(&result)
	if err != nil {
		if !strings.Contains(err.Error(), "no documents in result") {
			log.Error("Store mongo Get %s: %s", key, err.Error())
		}
		return nil, false
	}
	value, has := result["value"]
	return value, has
}

// Set adds a value to the store.
func (store *Store) Set(key string, value interface{}, ttl time.Duration) error {
	filter := bson.D{{Key: "key", Value: key}}
	doc := bson.D{{Key: "key", Value: key}, {Key: "value", Value: value}}
	opts := options.FindOneAndReplace().SetUpsert(true)
	res := store.Collection.FindOneAndReplace(context.TODO(), filter, doc, opts)
	err := res.Err()
	if err != nil && !strings.Contains(err.Error(), "no documents in result") {
		log.Error("Store mongo Set %s: %s", key, res.Err().Error())
		return err
	}

	return nil
}

// Del remove is used to purge a key from the store
func (store *Store) Del(key string) error {
	filter := bson.D{{Key: "key", Value: key}}
	_, err := store.Collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo Del: %s", err.Error())
		return err
	}
	return nil
}

// Has check if the store is exist ( without updating recency or frequency )
func (store *Store) Has(key string) bool {
	filter := bson.D{{Key: "key", Value: key}}
	result, err := store.Collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo Has: %s", err.Error())
		return false
	}

	return int(result) == 1
}

// Len returns the number of stored entries (**not O(1)**)
func (store *Store) Len() int {
	result, err := store.Collection.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		log.Error("Store mongo Has: %s", err.Error())
		return 0
	}
	return int(result)
}

// Keys returns all the cached keys
func (store *Store) Keys() []string {
	cursor, err := store.Collection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}

	keys := []string{}
	for cursor.Next(context.TODO()) {
		var result bson.D
		if err := cursor.Decode(&result); err != nil {
			log.Error("Store mongo Keys: %s", err.Error())
			continue
		}
		keys = append(keys, fmt.Sprintf("%s", result.Map()["key"]))
	}

	return keys
}

// Clear is used to clear the cache
func (store *Store) Clear() {
	keys := store.Keys()
	for _, key := range keys {
		store.Del(key)
	}
}

// GetSet looks up a key's value from the cache. if does not exist add to the cache
func (store *Store) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	value, ok := store.Get(key)
	if !ok {
		var err error
		value, err = getValue(key)
		if err != nil {
			return nil, err
		}
		store.Set(key, value, ttl)
	}
	return value, nil
}

// GetDel looks up a key's value from the cache, then remove it.
func (store *Store) GetDel(key string) (value interface{}, ok bool) {
	value, ok = store.Get(key)
	if !ok {
		return nil, false
	}
	err := store.Del(key)
	if err != nil {
		return value, false
	}
	return value, true
}

// GetMulti mulit get values
func (store *Store) GetMulti(keys []string) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, _ := store.Get(key)
		values[key] = value
	}
	return values
}

// SetMulti mulit set values
func (store *Store) SetMulti(values map[string]interface{}, ttl time.Duration) {
	for key, value := range values {
		store.Set(key, value, ttl)
	}
}

// DelMulti mulit remove values
func (store *Store) DelMulti(keys []string) {
	for _, key := range keys {
		store.Del(key)
	}
}

// GetSetMulti mulit get values, if does not exist add to the cache
func (store *Store) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, ok := store.Get(key)
		if !ok {
			var err error
			value, err = getValue(key)
			if err != nil {
				log.Error("GetSetMulti Set %s: %s", key, err.Error())
			}
		}
		values[key] = value
	}
	return values
}
