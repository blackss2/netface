package rethinkdb

import (
	"strings"

	"github.com/blackss2/netface/pkg/store"

	"github.com/blackss2/utility/convert"
	"github.com/dancannon/gorethink"
)

func createDB(session *gorethink.Session, dbName string) (gorethink.Term, error) {
	if true {
		hasDB := false
		res, err := gorethink.DBList().Run(session)
		if err != nil {
			return gorethink.Term{}, err
		}
		defer res.Close()

		var row interface{}
		for res.Next(&row) {
			if convert.String(row) == dbName {
				hasDB = true
			}
		}
		if !hasDB {
			gorethink.DBCreate(dbName).Run(session)
		}
	}
	return gorethink.DB(dbName), nil
}

func createTable(session *gorethink.Session, db gorethink.Term, tableName string, tableCreate bool, indexNames []string, indexCreate bool, indexDelete bool) (gorethink.Term, error) {
	if true {
		thash, err := tableHash(session, db)
		if err != nil {
			return gorethink.Term{}, err
		}
		hasTable := thash[tableName]
		if !hasTable && !tableCreate {
			return gorethink.Term{}, store.ErrNotExistTable
		}
		indexCreateTarget := make([]string, 0, len(indexNames))
		indexDeleteTarget := make([]string, 0, len(indexNames))
		if hasTable {
			if true {
				ihash, err := indexHash(session, db, tableName)
				if err != nil {
					return gorethink.Term{}, err
				}
				for _, v := range indexNames {
					if ihash[v] {
						delete(ihash, v)
					} else {
						indexCreateTarget = append(indexCreateTarget, v)
					}
				}
				for k, _ := range ihash {
					indexDeleteTarget = append(indexDeleteTarget, k)
				}
			}
			if !indexCreate && len(indexCreateTarget) > 0 {
				return gorethink.Term{}, store.ErrNotExistIndex
			}
			if !indexDelete && len(indexDeleteTarget) > 0 {
				return gorethink.Term{}, store.ErrNotPermitted
			}
		} else {
			_, err := db.TableCreate(tableName).Run(session)
			if err != nil {
				return gorethink.Term{}, err
			}
			for _, v := range indexNames {
				indexCreateTarget = append(indexCreateTarget, v)
			}
		}
		for _, v := range indexCreateTarget {
			if strings.Index(v, ".") >= 0 {
				list := strings.Split(v, ".")
				fields := make([]interface{}, 0)
				for _, f := range list {
					fields = append(fields, gorethink.Row.Field(f))
				}

				err := db.Table(tableName).IndexCreateFunc(v, fields).Exec(session)
				if err != nil {
					return gorethink.Term{}, err
				}
			} else if strings.Index(v, "->") >= 0 {
				list := strings.Split(v, "->")
				var fields gorethink.Term
				for i, f := range list {
					if i == 0 {
						fields = gorethink.Row.Field(f)
					} else {
						fields = fields.Field(f)
					}
				}

				err := db.Table(tableName).IndexCreateFunc(v, fields).Exec(session)
				if err != nil {
					return gorethink.Term{}, err
				}
			} else {
				_, err := db.Table(tableName).IndexCreate(v).Run(session)
				if err != nil {
					return gorethink.Term{}, err
				}
			}
		}
		for _, v := range indexDeleteTarget {
			_, err := db.Table(tableName).IndexDrop(v).Run(session)
			if err != nil {
				return gorethink.Term{}, err
			}
		}
	}
	return db.Table(tableName), nil
}

func tableHash(session *gorethink.Session, db gorethink.Term) (map[string]bool, error) {
	res, err := db.TableList().Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	hash := make(map[string]bool)
	var row interface{}
	for res.Next(&row) {
		hash[convert.String(row)] = true
	}
	return hash, nil
}

func indexHash(session *gorethink.Session, db gorethink.Term, tableName string) (map[string]bool, error) {
	res, err := db.Table(tableName).IndexList().Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	hash := make(map[string]bool, 0)
	var row interface{}
	for res.Next(&row) {
		hash[convert.String(row)] = true
	}
	return hash, nil
}
