package mongo

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	mongodb "github.com/yaoapp/gou/connector/mongo"
	"github.com/yaoapp/kun/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// New create a new store via connector
func New(c connector.Connector) (*Store, error) {
	mongodb, ok := c.(*mongodb.Connector)
	if !ok {
		return nil, fmt.Errorf("the connector was not a *mongodb.Connector")
	}

	// coll
	coll := mongodb.Database.Collection(mongodb.ID())

	// Create unique index on key
	keyIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: -1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := coll.Indexes().CreateOne(context.TODO(), keyIndex)
	if err != nil {
		return nil, err
	}

	// Create TTL index on expired_at field
	// MongoDB will automatically delete documents when expired_at time is reached
	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expired_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0), // 0 means delete at the exact expired_at time
	}

	_, err = coll.Indexes().CreateOne(context.TODO(), ttlIndex)
	if err != nil {
		// TTL index might already exist, log warning but don't fail
		log.Warn("Store mongo TTL index: %s", err.Error())
	}

	return &Store{Database: mongodb.Database, Collection: coll, Option: Option{Prefix: mongodb.ID() + ":"}}, nil
}

// NewWithOption create a new store via connector with options
func NewWithOption(c connector.Connector, opt Option) (*Store, error) {
	store, err := New(c)
	if err != nil {
		return nil, err
	}
	store.Option = opt
	return store, nil
}

// prefixKey adds the prefix to a key
func (store *Store) prefixKey(key string) string {
	if store.Option.Prefix == "" {
		return key
	}
	return store.Option.Prefix + key
}

// unprefixKey removes the prefix from a key
func (store *Store) unprefixKey(key string) string {
	if store.Option.Prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, store.Option.Prefix)
}

// convertValue handles primitive.Binary conversion to []byte for consistency with other stores
func convertValue(value interface{}) interface{} {
	if binary, ok := value.(primitive.Binary); ok {
		return binary.Data
	}
	return value
}

// convertSlice handles primitive.Binary conversion for slices
func convertSlice(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, item := range slice {
		result[i] = convertValue(item)
	}
	return result
}

// Get looks up a key's value from the store.
func (store *Store) Get(key string) (value interface{}, ok bool) {
	prefixedKey := store.prefixKey(key)
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: prefixedKey}}).Decode(&result)
	if err != nil {
		if !strings.Contains(err.Error(), "no documents in result") {
			log.Error("Store mongo Get %s: %s", prefixedKey, err.Error())
		}
		return nil, false
	}

	// Check if expired for immediate consistency
	if expiredAt, has := result["expired_at"]; has && expiredAt != nil {
		if t, ok := expiredAt.(primitive.DateTime); ok {
			if time.Now().After(t.Time()) {
				// Expired, delete and return not found
				store.Del(key)
				return nil, false
			}
		}
	}

	value, has := result["value"]
	if !has {
		return nil, false
	}

	// Handle primitive.Binary by converting to []byte for consistency with other stores
	return convertValue(value), true
}

// Set adds a value to the store.
func (store *Store) Set(key string, value interface{}, ttl time.Duration) error {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}

	// Build document with optional TTL
	doc := bson.D{{Key: "key", Value: prefixedKey}, {Key: "value", Value: value}}
	if ttl > 0 {
		expiredAt := time.Now().Add(ttl)
		doc = append(doc, bson.E{Key: "expired_at", Value: expiredAt})
	} else {
		// No TTL - set expired_at to nil to clear any previous expiration
		doc = append(doc, bson.E{Key: "expired_at", Value: nil})
	}

	opts := options.FindOneAndReplace().SetUpsert(true)
	res := store.Collection.FindOneAndReplace(context.TODO(), filter, doc, opts)
	err := res.Err()
	if err != nil && !strings.Contains(err.Error(), "no documents in result") {
		log.Error("Store mongo Set %s: %s", prefixedKey, res.Err().Error())
		return err
	}

	return nil
}

// Del remove is used to purge a key from the store
// Supports wildcard pattern with * (e.g., "user:123:*")
func (store *Store) Del(key string) error {
	// Check if key contains wildcard
	if strings.Contains(key, "*") {
		return store.delPattern(key)
	}
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	_, err := store.Collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo Del: %s", err.Error())
		return err
	}
	return nil
}

// delPattern deletes all keys matching the pattern using regex
func (store *Store) delPattern(pattern string) error {
	// Add prefix to pattern
	fullPattern := store.prefixKey(pattern)
	// Convert wildcard pattern to regex
	// e.g., "user:123:*" -> "^user:123:.*"
	regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(fullPattern), "\\*", ".*") + "$"

	filter := bson.D{{Key: "key", Value: bson.D{{Key: "$regex", Value: regexPattern}}}}
	_, err := store.Collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo delPattern %s: %s", fullPattern, err.Error())
		return err
	}
	return nil
}

// Has check if the store is exist ( without updating recency or frequency )
func (store *Store) Has(key string) bool {
	// Use Get to check existence (includes TTL check)
	_, ok := store.Get(key)
	return ok
}

// Len returns the number of stored entries (**not O(1)**)
// Optional pattern parameter supports * wildcard (e.g., "user:*")
func (store *Store) Len(pattern ...string) int {
	// Build full pattern with prefix
	var fullPattern string
	if len(pattern) > 0 && pattern[0] != "" {
		fullPattern = store.prefixKey(pattern[0])
	} else if store.Option.Prefix != "" {
		fullPattern = store.Option.Prefix + "*"
	}

	// Build filter with non-expired documents
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "expired_at", Value: nil}},
			bson.D{{Key: "expired_at", Value: bson.D{{Key: "$exists", Value: false}}}},
			bson.D{{Key: "expired_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
		}},
	}

	// Add pattern filter if provided
	if fullPattern != "" {
		regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(fullPattern), "\\*", ".*") + "$"
		filter = append(filter, bson.E{Key: "key", Value: bson.D{{Key: "$regex", Value: regexPattern}}})
	}

	result, err := store.Collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo Len: %s", err.Error())
		return 0
	}
	return int(result)
}

// Keys returns all the cached keys
// Optional pattern parameter supports * wildcard (e.g., "user:*")
func (store *Store) Keys(pattern ...string) []string {
	// Build full pattern with prefix
	var fullPattern string
	if len(pattern) > 0 && pattern[0] != "" {
		fullPattern = store.prefixKey(pattern[0])
	} else if store.Option.Prefix != "" {
		fullPattern = store.Option.Prefix + "*"
	}

	// Build filter with non-expired documents
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "expired_at", Value: nil}},
			bson.D{{Key: "expired_at", Value: bson.D{{Key: "$exists", Value: false}}}},
			bson.D{{Key: "expired_at", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
		}},
	}

	// Add pattern filter if provided
	if fullPattern != "" {
		regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(fullPattern), "\\*", ".*") + "$"
		filter = append(filter, bson.E{Key: "key", Value: bson.D{{Key: "$regex", Value: regexPattern}}})
	}

	cursor, err := store.Collection.Find(context.TODO(), filter)
	if err != nil {
		log.Error("Store mongo Keys: %s", err.Error())
		return []string{}
	}
	defer cursor.Close(context.TODO())

	prefixLen := len(store.Option.Prefix)
	keys := []string{}
	for cursor.Next(context.TODO()) {
		var result bson.D
		if err := cursor.Decode(&result); err != nil {
			log.Error("Store mongo Keys: %s", err.Error())
			continue
		}
		key := fmt.Sprintf("%s", result.Map()["key"])
		// Remove prefix from returned keys
		if prefixLen > 0 && len(key) >= prefixLen {
			key = key[prefixLen:]
		}
		keys = append(keys, key)
	}

	return keys
}

// Clear is used to clear the cache
// If prefix is set, only clears keys with that prefix
func (store *Store) Clear() {
	if store.Option.Prefix != "" {
		// Only delete keys with the prefix
		store.Del("*")
	} else {
		keys := store.Keys()
		for _, key := range keys {
			store.Del(key)
		}
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
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "value", Value: bson.D{{Key: "$each", Value: values}}}}}}
	opts := options.Update().SetUpsert(true)

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Error("Store mongo Push %s: %s", prefixedKey, err.Error())
		return err
	}
	return nil
}

// Pop removes and returns an element from a list using MongoDB $pop operator
func (store *Store) Pop(key string, position int) (interface{}, error) {
	prefixedKey := store.prefixKey(key)
	// First get the value to return
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: prefixedKey}}).Decode(&result)
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
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$pop", Value: bson.D{{Key: "value", Value: popDirection}}}}

	_, err = store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo Pop %s: %s", key, err.Error())
		return nil, err
	}

	return convertValue(popValue), nil
}

// Pull removes all occurrences of a value from a list using MongoDB $pull operator
func (store *Store) Pull(key string, value interface{}) error {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$pull", Value: bson.D{{Key: "value", Value: value}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo Pull %s: %s", prefixedKey, err.Error())
		return err
	}
	return nil
}

// PullAll removes all occurrences of multiple values from a list using MongoDB $pullAll operator
func (store *Store) PullAll(key string, values []interface{}) error {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$pullAll", Value: bson.D{{Key: "value", Value: values}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo PullAll %s: %s", prefixedKey, err.Error())
		return err
	}
	return nil
}

// AddToSet adds values to a list only if they don't already exist using MongoDB $addToSet operator
func (store *Store) AddToSet(key string, values ...interface{}) error {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$addToSet", Value: bson.D{{Key: "value", Value: bson.D{{Key: "$each", Value: values}}}}}}
	opts := options.Update().SetUpsert(true)

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Error("Store mongo AddToSet %s: %s", prefixedKey, err.Error())
		return err
	}
	return nil
}

// ArrayLen returns the length of a list
func (store *Store) ArrayLen(key string) int {
	prefixedKey := store.prefixKey(key)
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: prefixedKey}}}},
		bson.D{{Key: "$project", Value: bson.D{{Key: "length", Value: bson.D{{Key: "$size", Value: "$value"}}}}}},
	}

	cursor, err := store.Collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		log.Error("Store mongo ArrayLen %s: %s", prefixedKey, err.Error())
		return 0
	}
	defer cursor.Close(context.TODO())

	var result struct {
		Length int `bson:"length"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			log.Error("Store mongo ArrayLen decode %s: %s", prefixedKey, err.Error())
			return 0
		}
		return result.Length
	}

	return 0
}

// ArrayGet returns an element at the specified index
func (store *Store) ArrayGet(key string, index int) (interface{}, error) {
	prefixedKey := store.prefixKey(key)
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: prefixedKey}}}},
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
		return convertValue(result.Element), nil
	}

	return nil, fmt.Errorf("index out of range")
}

// ArraySet sets an element at the specified index
func (store *Store) ArraySet(key string, index int, value interface{}) error {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: fmt.Sprintf("value.%d", index), Value: value}}}}

	_, err := store.Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Error("Store mongo ArraySet %s[%d]: %s", prefixedKey, index, err.Error())
		return err
	}
	return nil
}

// ArraySlice returns a slice of the list using MongoDB $slice operator
func (store *Store) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	prefixedKey := store.prefixKey(key)
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "key", Value: prefixedKey}}}},
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
		return convertSlice(result.Slice), nil
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
	prefixedKey := store.prefixKey(key)
	var result bson.M
	err := store.Collection.FindOne(context.TODO(), bson.D{{Key: "key", Value: prefixedKey}}).Decode(&result)
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
		return convertSlice(interfaceList), nil
	}

	return nil, fmt.Errorf("key is not a list")
}

// Incr increments a numeric value and returns the new value
func (store *Store) Incr(key string, delta int64) (int64, error) {
	prefixedKey := store.prefixKey(key)
	filter := bson.D{{Key: "key", Value: prefixedKey}}
	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "value", Value: delta}}}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result bson.M
	err := store.Collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&result)
	if err != nil {
		log.Error("Store mongo Incr %s: %s", prefixedKey, err.Error())
		return 0, err
	}

	if value, has := result["value"]; has {
		return toInt64(value), nil
	}
	return delta, nil
}

// Decr decrements a numeric value and returns the new value
func (store *Store) Decr(key string, delta int64) (int64, error) {
	return store.Incr(key, -delta)
}

// toInt64 converts an interface{} to int64
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}
