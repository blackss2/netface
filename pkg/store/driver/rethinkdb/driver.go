package rethinkdb

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/blackss2/netface/pkg/store"

	"github.com/dancannon/gorethink"
)

var gSessionPool = make(map[string]*sessionRetain)
var gSessionPoolMutex sync.Mutex

func init() {
	store.Register("rethinkdb", &Driver{})
	gorethink.SetTags("gorethink", "json")
}

type Driver struct {
	address string
	session *gorethink.Session
	db      gorethink.Term
	table   gorethink.Term
}

func (dr *Driver) Connect(conn string) error {
	err := dr.Close()
	if err != nil {
		return err
	}

	list := strings.Split(conn, "/")
	if len(list) != 2 {
		return store.ErrInvalidConnectionString
	}

	addr := list[0]
	database := list[1]

	var session *gorethink.Session
	gSessionPoolMutex.Lock()
	if s, has := gSessionPool[addr]; has {
		s.Retain++
		session = s.Session
	} else {
		sn, err := gorethink.Connect(gorethink.ConnectOpts{
			Address:    addr,
			InitialCap: 10,
			MaxOpen:    10,
		})
		if err != nil {
			return err
		}
		gSessionPool[addr] = &sessionRetain{
			Session: sn,
			Retain:  1,
		}
		session = sn
	}
	gSessionPoolMutex.Unlock()
	dr.address = addr
	dr.session = session

	runtime.SetFinalizer(dr, func(v interface{}) {
		v.(*Driver).Close()
	})

	db, err := createDB(session, database)
	if err != nil {
		return dr.Close()
	}
	dr.db = db

	return nil
}

func (dr *Driver) Close() error {
	var err error
	if dr.session != nil {
		gSessionPoolMutex.Lock()
		if s, has := gSessionPool[dr.address]; has {
			s.Retain--
			if s.Retain <= 0 {
				delete(gSessionPool, dr.address)
				err = dr.session.Close()
			}
		}
		gSessionPoolMutex.Unlock()
		dr.session = nil
	}
	return err
}

func (dr *Driver) Truncate() error {
	_, err := dr.table.Delete().RunWrite(dr.session)
	return err
}

func (dr *Driver) InitTable(opt store.TableOption) error {
	table, err := createTable(dr.session, dr.db, opt.TableName, opt.TableCreate, opt.IndexNames, opt.IndexCreate, opt.IndexDelete)
	if err != nil {
		return err
	}
	dr.table = table
	return nil
}
func (dr *Driver) Get(Id string, value interface{}) error {
	res, err := dr.table.Get(Id).Run(dr.session)
	if err != nil {
		return err
	}
	defer res.Close()

	err = res.One(value)
	if err != nil {
		if err == gorethink.ErrEmptyResult {
			return store.ErrNotExist
		} else {
			return err
		}
	}

	return nil
}

func (dr *Driver) Count(opts ...store.ListOption) (int64, error) {
	t := dr.table
	for _, opt := range opts {
		t = applyWhereOption(t, opt.WhereOption)
	}
	t = t.Count()

	res, err := t.Run(dr.session)
	if err != nil {
		return 0, err
	}
	defer res.Close()

	var count int64
	err = res.One(&count)
	if err != nil {
		if err == gorethink.ErrEmptyResult {
			if err == gorethink.ErrEmptyResult {
				return 0, store.ErrNotExist
			} else {
				return 0, err
			}
		} else {
			return 0, err
		}
	}

	return count, nil
}

func (dr *Driver) List(list interface{}, opts ...store.ListOption) error {
	t := dr.table
	Fields := make([]string, 0)
	for _, opt := range opts {
		t = applyWhereOption(t, opt.WhereOption)
		if len(opt.OrderAsc) > 0 {
			values := make([]interface{}, len(opt.OrderAsc))
			for i, v := range opt.OrderAsc {
				values[i] = v
			}
			t = t.OrderBy(values...)
		}
		if len(opt.OrderDesc) > 0 {
			values := make([]interface{}, len(opt.OrderDesc))
			for i, v := range opt.OrderDesc {
				values[i] = v
			}
			t = t.OrderBy(gorethink.Desc(values...))
		}
		if opt.Offset > 0 {
			t = t.Skip(opt.Offset)
		}
		if opt.Limit > 0 {
			t = t.Limit(opt.Limit)
		}
		if len(opt.Fields) > 0 {
			Fields = append(Fields, opt.Fields...)
		}
	}

	if len(Fields) > 0 {
		t = t.Pluck(Fields)
	}

	res, err := t.Run(dr.session)
	if err != nil {
		return err
	}
	defer res.Close()

	err = res.All(list)
	if err != nil {
		if err == gorethink.ErrEmptyResult {
			v := reflect.ValueOf(list)
			if v.Kind() == reflect.Ptr {
				v.Elem().Set(reflect.ValueOf(nil))
			} else {
				return store.ErrInvalidArgument
			}
		} else {
			return err
		}
	}

	return nil
}

func (dr *Driver) Insert(value interface{}) (string, error) {
	res, err := dr.table.
		Insert(value).
		RunWrite(dr.session)
	if err != nil {
		return "", err
	}

	if res.Inserted == 0 {
		return "", errors.New("not inserted")
	}

	if len(res.GeneratedKeys) == 0 {
		//id is specifieid
		return "", nil
	} else {
		return res.GeneratedKeys[0], nil
	}
}

func (dr *Driver) InsertBatch(value interface{}) ([]string, error) {
	res, err := dr.table.
		Insert(value).
		RunWrite(dr.session, gorethink.RunOpts{MaxBatchRows: 200})
	if err != nil {
		return nil, err
	}

	return res.GeneratedKeys, nil
}

func (dr *Driver) Update(Id string, value interface{}) error {
	//update data & time
	res, err := dr.table.
		Get(Id).
		Replace(value).
		RunWrite(dr.session)
	if err != nil {
		return err
	}

	if res.Unchanged == 0 && res.Updated == 0 && res.Replaced == 0 {
		return store.ErrNotExist
	}

	return nil
}

func (dr *Driver) Delete(Id string) error {
	res, err := dr.table.
		Get(Id).
		Delete().
		RunWrite(dr.session)
	if err != nil {
		return err
	}

	if res.Deleted == 0 {
		return store.ErrNotExist
	}

	return nil
}

func (dr *Driver) DeleteBatch(Ids []string) error {
	values := make([]interface{}, len(Ids))
	for i, v := range Ids {
		values[i] = v
	}

	res, err := dr.table.
		Get(values...).
		Delete().
		RunWrite(dr.session)
	if err != nil {
		return err
	}

	if res.Deleted == 0 {
		return store.ErrNotExist
	}

	return nil
}

func (dr *Driver) DeleteBy(opts ...store.DeleteOption) error {
	t := dr.table
	for _, opt := range opts {
		t = applyWhereOption(t, opt.WhereOption)
	}

	_, err := t.Delete().RunWrite(dr.session)
	if err != nil {
		return err
	}
	return nil
}

func applyWhereOption(t gorethink.Term, opt store.WhereOption) gorethink.Term {
	if opt.Not {
		if len(opt.IndexBy) > 0 {
			panic("not support IndexBy at NotWhere")
		}
		if len(opt.FieldBy) > 0 {
			if len(opt.FieldByValues) > 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return gorethink.Expr(opt.FieldByValues).Contains(row.Field(opt.FieldBy)).Not()
				})
			}
			if len(opt.FieldByLike) > 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return row.Field(opt.FieldBy).Match("(?i)^.*" + opt.FieldByLike + ".*$").Not()
				})
			}
		}
		if len(opt.Ids) > 0 {
			if len(opt.IndexBy) == 0 && len(opt.FieldBy) == 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return gorethink.Expr(opt.Ids).Contains(row.Field("id")).Not()
				})
			} else {
				t = t.Get(opt.Ids)
			}
		}
	} else {
		if len(opt.IndexBy) > 0 {
			if len(opt.IndexByValues) > 0 {
				t = t.GetAllByIndex(opt.IndexBy, opt.IndexByValues...)
			} else {
				t = t.GetAllByIndex(opt.IndexBy)
			}
		}
		if len(opt.FieldBy) > 0 {
			if len(opt.FieldByValues) > 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return gorethink.Expr(opt.FieldByValues).Contains(row.Field(opt.FieldBy))
				})
			}
			if len(opt.FieldByLike) > 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return row.Field(opt.FieldBy).Match("(?i)^.*" + opt.FieldByLike + ".*$")
				})
			}
		}
		if len(opt.Ids) > 0 {
			if len(opt.IndexBy) == 0 && len(opt.FieldBy) == 0 {
				t = t.Filter(func(row gorethink.Term) gorethink.Term {
					return gorethink.Expr(opt.Ids).Contains(row.Field("id"))
				})
			} else {
				t = t.Get(opt.Ids)
			}
		}
	}
	return t
}

type sessionRetain struct {
	Session *gorethink.Session
	Retain  int
}
