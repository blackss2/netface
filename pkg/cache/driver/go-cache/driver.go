package gocache

import (
	"encoding/json"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/blackss2/netface/pkg/cache"

	gocache "github.com/patrickmn/go-cache"
	"github.com/satori/go.uuid"
)

var gSessionPool = make(map[string]*sessionRetain)
var gSessionPoolMutex sync.Mutex

func init() {
	cache.Register("go-cache", &Driver{})
}

type Driver struct {
	address string
	session *gocache.Cache
	policy  cache.PolicyOption
}

func (dr *Driver) Connect(addr string, defaultPolicy cache.PolicyOption) error {
	err := dr.Close()
	if err != nil {
		return err
	}

	if addr != "localhost" {
		return cache.ErrInvalidHost
	}

	var session *gocache.Cache
	gSessionPoolMutex.Lock()
	if s, has := gSessionPool[addr]; has {
		s.Retain++
		session = s.Session
	} else {
		cs := gocache.New(defaultPolicy.DefaultExpiration, defaultPolicy.PurgeInterval)
		gSessionPool[addr] = &sessionRetain{
			Session: cs,
			Retain:  1,
		}
		session = cs
	}
	gSessionPoolMutex.Unlock()
	dr.address = addr
	dr.session = session
	dr.policy = defaultPolicy

	runtime.SetFinalizer(dr, func(v interface{}) {
		v.(*Driver).Close()
	})

	return nil
}

func (dr *Driver) Close() error {
	if dr.session != nil {
		gSessionPoolMutex.Lock()
		if s, has := gSessionPool[dr.address]; has {
			s.Retain--
			if s.Retain <= 0 {
				delete(gSessionPool, dr.address)
				dr.session.Flush()
			}
		}
		gSessionPoolMutex.Unlock()
		dr.session = nil
	}
	return nil
}

func (dr *Driver) Truncate() error {
	dr.session.Flush()
	return nil
}

func (dr *Driver) Get(Id string, value interface{}) error {
	v, has := dr.session.Get(Id)
	if !has {
		return cache.ErrNotExist
	}

	err := json.Unmarshal(v.([]byte), value)
	if err != nil {
		return err
	}
	return nil
}

func (dr *Driver) IsExist(Id string) (bool, error) {
	_, has := dr.session.Get(Id)
	return has, nil
}

func (dr *Driver) Set(Id string, value interface{}) error {
	return dr.SetWith(Id, value, cache.SetterOption{Expiration: dr.policy.DefaultExpiration})
}

func (dr *Driver) SetWith(Id string, value interface{}, opt cache.SetterOption) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiration time.Duration
	if opt.IsNoExpiration {
		expiration = gocache.NoExpiration
	} else {
		expiration = opt.Expiration
	}

	dr.session.Set(Id, data, expiration)
	return nil
}

func (dr *Driver) Insert(value interface{}) (string, error) {
	return dr.InsertWith(value, cache.SetterOption{Expiration: dr.policy.DefaultExpiration})
}

func (dr *Driver) InsertWith(value interface{}, opt cache.SetterOption) (string, error) {
	id := uuid.NewV1().String()

	err := dr.SetWith(id, value, opt)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (dr *Driver) NextSequence(Id string) (uint64, error) {
	key := "_seq_" + Id
	value, err := dr.session.IncrementUint64(key, 1)
	if err != nil {
		if strings.HasSuffix(err.Error(), "not found") {
			value = 1
			dr.session.Set(key, value, gocache.NoExpiration)
		} else {
			return 0, err
		}
	}
	return value, nil
}

func (dr *Driver) Delete(Id string) error {
	dr.session.Delete(Id)
	return nil
}

type sessionRetain struct {
	Session *gocache.Cache
	Retain  int
}
