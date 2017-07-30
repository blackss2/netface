package cache

import (
	"reflect"
	"time"
)

var gCacheHash = make(map[string]reflect.Type)

func Register(name string, driver Cache) {
	gCacheHash[name] = reflect.TypeOf(driver).Elem()
}

func NewCache(name string) (Cache, error) {
	if t, has := gCacheHash[name]; !has {
		return nil, ErrNotSupportedDriver
	} else {
		return reflect.New(t).Interface().(Cache), nil
	}
}

type Cache interface {
	Connect(Addr string, defaultPolicy PolicyOption) error
	Close() error
	Truncate() error
	Get(Id string, value interface{}) error
	IsExist(Id string) (bool, error)
	Set(Id string, value interface{}) error
	SetWith(Id string, value interface{}, opt SetterOption) error
	Insert(value interface{}) (string, error)
	InsertWith(value interface{}, opt SetterOption) (string, error)
	NextSequence(Id string) (uint64, error)
	Delete(Id string) error
}

type PolicyOption struct {
	DefaultExpiration time.Duration
	PurgeInterval     time.Duration
}

type SetterOption struct {
	Expiration     time.Duration
	IsNoExpiration bool
}
