package gocache

import (
	"bytes"
	"encoding/gob"
	"runtime"
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

func (dr *Driver) Register(Addr string, Name string) (string, error) {
	return "", nil
}

func (dr *Driver) Unregster(Id string) error {
	return nil
}

func (dr *Driver) Ping(Id string) error {
	return nil
}

type sessionRetain struct {
	Session *gocache.Cache
	Retain  int
}
