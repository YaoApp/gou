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

// Push adds values to the end of a list using MongoDB $push operator
func (store *Store) Push(key string, values ...interface{}) error {
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "value", Value: bson.D{{Key: "$each", Value: values}}}}}}
	opts := options.Update().SetUpsert(true)

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Error("Store mongo Push %s: %s", key, err.Error())
		return err
	}
	return nil
}

// Pop removes and returns an element from a list using MongoDB $pop operator
func (store *Store) Pop(key string, position int) (interface{}, error) {
	// First get the value to return
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: key}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("key not found")
	}

	value, has := result["value"]
	if !has {
		return nil, fmt.Errorf("key not found")
	}

	list, ok := value.(bson.A)
	if !ok {
		return nil, fmt.Errorf("key is not a list")
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("list is empty")
	}

	var popValue interface{}
	var popDirection int
	if position == 1 { // pop from end
		popValue = list[len(list)-1]
		popDirection = 1
	} else { // pop from beginning
		popValue = list[0]
		popDirection = -1
	}

	// Use MongoDB $pop operator
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$pop", Value: bson.D{{Key: "value", Value: popDirection}}}}

	_, err = store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo Pop %s: %s", key, err.Error())
		return nil, err
	}

	return popValue, nil
}

// Pull removes all occurrences of a value from a list using MongoDB $pull operator
func (store *Store) Pull(key string, value interface{}) error {
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$pull", Value: bson.D{{Key: "value", Value: value}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo Pull %s: %s", key, err.Error())
		return err
	}
	return nil
}

// PullAll removes all occurrences of multiple values from a list using MongoDB $pullAll operator
func (store *Store) PullAll(key string, values []interface{}) error {
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$pullAll", Value: bson.D{{Key: "value", Value: values}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo PullAll %s: %s", key, err.Error())
		return err
	}
	return nil
}

// AddToSet adds values to a list only if they don't already exist using MongoDB $addToSet operator
func (store *Store) AddToSet(key string, values ...interface{}) error {
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$addToSet", Value: bson.D{{Key: "value", Value: bson.D{{Key: "$each", Value: values}}}}}}
	opts := options.Update().SetUpsert(true)

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Error("Store mongo AddToSet %s: %s", key, err.Error())
		return err
	}
	return nil
}

// ArrayLen returns the length of a list
func (store *Store) ArrayLen(key string) int {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: key}}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "length", Value: bson.D{{Key: "$size", Value: "$value"}}}}}},
	}

	cursor, err := store.Collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		log.Error("Store mongo ArrayLen %s: %s", key, err.Error())
		return 0
	}
	defer cursor.Close(context.TODO())

	var result struct {
		Length int `bson:"length"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			log.Error("Store mongo ArrayLen decode %s: %s", key, err.Error())
			return 0
		}
		return result.Length
	}

	return 0
}

// ArrayGet returns an element at the specified index
func (store *Store) ArrayGet(key string, index int) (interface{}, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: key}}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "element", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$value", index}}}}}}},
	}

	cursor, err := store.Collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var result struct {
		Element interface{} `bson:"element"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return result.Element, nil
	}

	return nil, fmt.Errorf("index out of range")
}

// ArraySet sets an element at the specified index
func (store *Store) ArraySet(key string, index int, value interface{}) error {
	filter := bson.D{{Key: "key", Value: key}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: fmt.Sprintf("value.%d", index), Value: value}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo ArraySet %s[%d]: %s", key, index, err.Error())
		return err
	}
	return nil
}

// ArraySlice returns a slice of the list using MongoDB $slice operator
func (store *Store) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: key}}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "slice", Value: bson.D{{Key: "$slice", Value: bson.A{"$value", skip, limit}}}}}}},
	}

	cursor, err := store.Collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var result struct {
		Slice []interface{} `bson:"slice"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return result.Slice, nil
	}

	return []interface{}{}, nil
}

// ArrayPage returns a page of the list
func (store *Store) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	if page < 1 || pageSize < 1 {
		return []interface{}{}, nil
	}

	skip := (page - 1) * pageSize
	return store.ArraySlice(key, skip, pageSize)
}

// ArrayAll returns all elements in the list
func (store *Store) ArrayAll(key string) ([]interface{}, error) {
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: key}}).Decode(&result)
	if err != nil {
		if strings.Contains(err.Error(), "no documents in result") {
			return []interface{}{}, nil
		}
		return nil, err
	}

	value, has := result["value"]
	if !has {
		return []interface{}{}, nil
	}

	if list, ok := value.(bson.A); ok {
		interfaceList := make([]interface{}, len(list))
		copy(interfaceList, list)
		return interfaceList, nil
	}

	return nil, fmt.Errorf("key is not a list")
}
