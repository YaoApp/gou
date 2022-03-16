package kv

// Store The interface of a key-value store
type Store interface {
	Get(key string) (value interface{}, ok bool)
	Set(key string, value interface{})
	Del(key string)
	Has(key string) bool
	Len() int
	Keys() []string
	Clear()
	GetSet(key string, getValue func(key string) (interface{}, error)) (interface{}, error)
	GetDel(key string) (value interface{}, ok bool)
	GetMulti(keys []string) map[string]interface{}
	SetMulti(values map[string]interface{})
	DelMulti(keys []string)
	GetSetMulti(keys []string, getValue func(key string) (interface{}, error)) map[string]interface{}
}
