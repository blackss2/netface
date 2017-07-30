package memcached

import (
	"encoding/binary"
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/blackss2/netface/pkg/cache"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/satori/go.uuid"
)

const (
	NoExpiration = time.Duration(0)
)

var gSessionPool = make(map[string]*sessionRetain)
var gSessionPoolMutex sync.Mutex

func init() {
	cache.Register("memcached", &Driver{})
}

type Driver struct {
	address string
	session *memcache.Client
	policy  cache.PolicyOption
}

func (dr *Driver) Connect(addr string, defaultPolicy cache.PolicyOption) error {
	err := dr.Close()
	if err != nil {
		return err
	}

	var session *memcache.Client
	gSessionPoolMutex.Lock()
	if s, has := gSessionPool[addr]; has {
		s.Retain++
		session = s.Session
	} else {
		cs := memcache.New(addr)
		_, err := cs.Get("__PING__")
		if err != nil {
			if err != memcache.ErrCacheMiss {
				return err
			}
		}
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
				dr.session.FlushAll()
			}
		}
		gSessionPoolMutex.Unlock()
		dr.session = nil
	}
	return nil
}

func (dr *Driver) Truncate() error {
	return dr.session.FlushAll()
}

func (dr *Driver) Get(Id string, value interface{}) error {
	v, err := dr.session.Get(Id)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return cache.ErrNotExist
		} else {
			return err
		}
	}

	err = json.Unmarshal([]byte(v.Value), value)
	if err != nil {
		return err
	}
	return nil
}

func (dr *Driver) IsExist(Id string) (bool, error) {
	_, err := dr.session.Get(Id)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
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
		expiration = NoExpiration
	} else {
		expiration = opt.Expiration
	}

	return dr.session.Set(&memcache.Item{
		Key:        Id,
		Value:      data,
		Expiration: int32(expiration / time.Second),
	})
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
	value, err := dr.session.Increment(key, 1)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			value = 1
			bs := make([]byte, 8)
			binary.LittleEndian.PutUint64(bs, value)
			err := dr.session.Set(&memcache.Item{
				Key:        key,
				Value:      bs,
				Expiration: 0,
			})
			if err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	return value, nil
}

func (dr *Driver) Delete(Id string) error {
	return dr.session.Delete(Id)
}

type sessionRetain struct {
	Session *memcache.Client
	Retain  int
}
