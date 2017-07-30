package registry

import (
	"reflect"
)

var gRegistryHash = make(map[string]reflect.Type)

func Register(name string, driver Registry) {
	gRegistryHash[name] = reflect.TypeOf(driver).Elem()
}

func NewRegistry(name string) (Registry, error) {
	if t, has := gRegistryHash[name]; !has {
		return nil, ErrNotSupportedDriver
	} else {
		return reflect.New(t).Interface().(Registry), nil
	}
}

type Registry interface {
	Connect(Url string) error
	Close() error
	Register(Addr string, Name string) (string, error)
	Unregister(Id string) error
	Ping(Id string) error
}
