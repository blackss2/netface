package store

import (
	"reflect"
)

var gStoreHash = make(map[string]reflect.Type)

func Register(name string, driver Store) {
	gStoreHash[name] = reflect.TypeOf(driver).Elem()
}

func NewStore(name string) (Store, error) {
	if t, has := gStoreHash[name]; !has {
		return nil, ErrNotSupportedDriver
	} else {
		return reflect.New(t).Interface().(Store), nil
	}
}

type Store interface {
	Connect(Url string) error
	Close() error
	InitTable(opt TableOption) error
	Truncate() error
	Get(Id string, value interface{}) error
	Count(opts ...ListOption) (int64, error)
	List(list interface{}, opts ...ListOption) error
	Insert(value interface{}) (string, error)
	InsertBatch(value interface{}) ([]string, error)
	Update(Id string, value interface{}) error
	Delete(Id string) error
	DeleteBatch(Ids []string) error
	DeleteBy(opts ...DeleteOption) error
}

type TableOption struct {
	TableName   string
	TableCreate bool
	IndexNames  []string
	IndexCreate bool
	IndexDelete bool
}

type WhereOption struct {
	Not           bool
	IndexBy       string
	IndexByValues []interface{}
	FieldBy       string
	FieldByLike   string
	FieldByValues []interface{}
	Ids           []string
}

type DeleteOption struct {
	WhereOption
}

type ListOption struct {
	WhereOption
	Fields    []string
	OrderAsc  []string
	OrderDesc []string
	Offset    int
	Limit     int
}

type QueryStore interface {
	Store
	Query(query string) (*QueryResult, error)
	BatchQuery(queryList []string) error
}

type QueryResult interface {
	Close() error
	IsNil() bool
	Columns() []string
	Next() bool
	FetchArray() []interface{}
	FetchHash() map[string]interface{}
	Fetch(interface{}) error
}
