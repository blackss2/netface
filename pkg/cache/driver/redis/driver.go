package redis

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/blackss2/netface/pkg/cache"

	"github.com/go-redis/redis"
	"github.com/satori/go.uuid"
)

const (
	NoExpiration = time.Duration(0)
)

var gSessionPool = make(map[string]*sessionRetain)
var gSessionPoolMutex sync.Mutex

func init() {
	cache.Register("redis", &Driver{})
}

type Driver struct {
	address string
	session *redis.Client
	policy  cache.PolicyOption
}

func (dr *Driver) Connect(addr string, defaultPolicy cache.PolicyOption) error {
	err := dr.Close()
	if err != nil {
		return err
	}

	var session *redis.Client
	gSessionPoolMutex.Lock()
	if s, has := gSessionPool[addr]; has {
		s.Retain++
		session = s.Session
	} else {
		cs := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "", // no password set
			DB:       0,
		})
		_, err := cs.Ping().Result()
		if err != nil {
			return err
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
				dr.session.FlushAll().Err()
			}
		}
		gSessionPoolMutex.Unlock()
		dr.session = nil
	}
	return nil
}

func (dr *Driver) Truncate() error {
	dr.session.FlushAll().Err()
	return nil
}

func (dr *Driver) Get(Id string, value interface{}) error {
	v, err := dr.session.Get(Id).Result()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrNotExist
		} else {
			return err
		}
	}

	err = json.Unmarshal([]byte(v), value)
	if err != nil {
		return err
	}
	return nil
}

func (dr *Driver) IsExist(Id string) (bool, error) {
	v, err := dr.session.Exists(Id).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		} else {
			return false, err
		}
	}
	return (v == 1), nil
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

	return dr.session.Set(Id, data, expiration).Err()
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
	value, err := dr.session.Incr(key).Result()
	if err != nil {
		if err == redis.Nil {
			value = 1
			err := dr.session.Set(key, value, NoExpiration).Err()
			if err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	return uint64(value), nil
}

func (dr *Driver) Delete(Id string) error {
	return dr.session.Del(Id).Err()
}

type sessionRetain struct {
	Session *redis.Client
	Retain  int
}
