package main

import (
	"log"
	"time"

	"github.com/blackss2/netface/pkg/cache"
	_ "github.com/blackss2/netface/pkg/cache/driver/go-cache"
	_ "github.com/blackss2/netface/pkg/cache/driver/memcached"
	_ "github.com/blackss2/netface/pkg/cache/driver/redis"
	"github.com/blackss2/netface/pkg/store"
	_ "github.com/blackss2/netface/pkg/store/driver/rethinkdb"
)

func main() {
	st, err := store.NewStore("rethinkdb")
	if err != nil {
		panic(err)
	}

	err = st.Connect("45.32.24.69:28015/testdb")
	if err != nil {
		panic(err)
	}

	err = st.InitTable(store.TableOption{
		TableName:   "test_table",
		TableCreate: true,
	})
	if err != nil {
		panic(err)
	}

	err = st.Truncate()
	if err != nil {
		panic(err)
	}

	id, err := st.Insert(map[string]interface{}{"111": true, "222": false})
	if err != nil {
		panic(err)
	}

	var abc map[string]interface{}
	err = st.Get(id, &abc)
	if err != nil {
		panic(err)
	}
	log.Println("store", abc)

	//c, err := cache.NewCache("go-cache")
	//c, err := cache.NewCache("redis")
	c, err := cache.NewCache("memcached")
	if err != nil {
		panic(err)
	}

	//err = c.Connect("localhost", cache.PolicyOption{
	//err = c.Connect("104.156.238.172:6379", cache.PolicyOption{
	err = c.Connect("104.156.238.172:11211", cache.PolicyOption{
		DefaultExpiration: time.Second * 1,
		PurgeInterval:     time.Second * 1,
	})
	if err != nil {
		panic(err)
	}

	log.Println(c.NextSequence("test"))
	log.Println(c.NextSequence("test"))
	log.Println(c.NextSequence("test2"))
	log.Println(c.NextSequence("test2"))
	log.Println(c.NextSequence("test2"))
	log.Println(c.NextSequence("test"))
	return

	if true {
		err = st.Get(id, &abc)
		if err != nil {
			panic(err)
		}
		log.Println("store", abc)

		err = c.Set("id", abc)
		if err != nil {
			panic(err)
		}
	}

	start := time.Now()
	for i := 0; i < 100; i++ {
		if true {
			err = c.Get("id", &abc)
			if err != nil {
				if err != cache.ErrNotExist {
					panic(err)
				}
				err = st.Get(id, &abc)
				if err != nil {
					panic(err)
				}
				log.Println("store", abc)
			} else {
				log.Println(abc)
			}
		}
	}
	log.Println(time.Now().Sub(start).Seconds())

	time.Sleep(time.Second * 2)

	start = time.Now()
	for i := 0; i < 100; i++ {
		if true {
			err = c.Get("id", &abc)
			if err != nil {
				if err != cache.ErrNotExist {
					panic(err)
				}
				err = st.Get(id, &abc)
				if err != nil {
					panic(err)
				}
				log.Println("store", abc)
			} else {
				log.Println(abc)
			}
		}
	}
	log.Println(time.Now().Sub(start).Seconds())
}
