package irwys

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DB structure.
type DB struct {
	db    *leveldb.DB
	batch *leveldb.Batch
	opts  *opt.Options
	lock  *sync.RWMutex
}

// NewDB creates an object of DB structure.
func NewDB(
	path string,
	name string,
	opts *opt.Options,
) DB {
	gob.Register(map[string]string{})
	lock := sync.RWMutex{}
	ldb, err := leveldb.OpenFile(filepath.Join(path, name), opts)
	if err != nil {
		Error.Println("Can't get access to database")
		panic(err)
	}
	db := DB{ldb, new(leveldb.Batch), opts, &lock}
	return db
}

// Get performs non-blocking get of an object from database.
func (db DB) Get(key string) (decoded interface{}, err error) {
	(*db.lock).RLock()
	defer (*db.lock).RUnlock()

	var ok bool
	var data []byte

	if ok, err = db.db.Has([]byte(key), nil); !ok {
		return
	}
	if data, err = db.db.Get([]byte(key), nil); err != nil {
		Error.Printf(
			"Can't get entry from DB\n\tKey: %s\n\tError: %s",
			key, err,
		)
	} else {
		d := gob.NewDecoder(bytes.NewReader(data))
		if err = d.Decode(&decoded); err != nil {
			Error.Printf("Can't decode value:\n\tValue: %s\n\tError: %s", data, err)
		}
	}

	return
}

// Put performs non-blocking put of an object into database.
func (db DB) Put(key string, value interface{}) (err error) {
	(*db.lock).Lock()
	defer (*db.lock).Unlock()

	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)

	if err = e.Encode(&value); err != nil {
		Error.Printf("Can't encode value:\n\tValue: %d\n\tError: %s", value, err)
	}

	if err = db.db.Put([]byte(key), b.Bytes(), nil); err != nil {
		Error.Printf(
			"Can't put entry to DB\n\tKey: %s\n\tValue: %d\n\tError: %s",
			key, value, err,
		)
	}

	return
}

// Delete performs non-blocking delete of an object from database.
func (db DB) Delete(key string) (err error) {
	(*db.lock).Lock()
	defer (*db.lock).Unlock()

	if err = db.db.Delete([]byte(key), nil); err != nil {
		Error.Printf(
			"Can't delete entry from DB\n\tKey: %s\n\tError: %s",
			key, err,
		)
	}

	return
}

// Exist performs non-blocking check of an object for existence in database.
func (db DB) Exist(key string) (exist bool, err error) {
	(*db.lock).RLock()
	defer (*db.lock).RUnlock()

	if exist, err = db.db.Has([]byte(key), nil); err != nil {
		Error.Printf(
			"Can't find entry in DB\n\tKey: %s\n\tError: %s",
			key, err,
		)
	}

	return
}

// BatchPut puts objects into the database by batching (leveldb).
func (db DB) BatchPut(key string, value string) {
	(*db.lock).Lock()
	defer (*db.lock).Unlock()

	db.batch.Put([]byte(key), []byte(value))
}

// BatchDelete deletes objects from the database by batching (leveldb).
func (db DB) BatchDelete(key string) {
	(*db.lock).Lock()
	defer (*db.lock).Unlock()

	db.batch.Delete([]byte(key))
}

// BatchWrite performs write of batch to database.
func (db DB) BatchWrite() (err error) {
	if err = db.db.Write(db.batch, nil); err != nil {
		Error.Printf("Can't write batch to DB\n\tError: %s", err)
	}
	db.batch.Reset()

	return err
}

// Iterate brings possibility to iterate over database.
func (db DB) Iterate(opts *util.Range) iterator.Iterator {
	return db.db.NewIterator(opts, nil)
}

// Close DB connection
func (db DB) Close() {
	db.db.Close()
}
