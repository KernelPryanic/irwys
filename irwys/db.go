package irwys

import (
	"bytes"
	"encoding/gob"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type DB struct {
	db   *leveldb.DB
	opts *opt.Options
}

func NewDB(
	path string,
	opts *opt.Options,
) DB {
	db, err := leveldb.OpenFile(path, opts)
	if err != nil {
		Error.Println("Can't get access to database")
		panic(err)
	}
	myDB := DB{db, opts}
	return myDB
}

func (myDB DB) Close() {
	myDB.db.Close()
}

func (myDB DB) Put(key string, value interface{}) (err error) {
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)

	if err = e.Encode(&value); err != nil {
		Error.Printf("Can't encode value:\n\tValue: %d\n\tError: %s", value, err)
	}

	if err = myDB.db.Put([]byte(key), b.Bytes(), nil); err != nil {
		Error.Printf(
			"Can't put entry to DB\n\tKey: %s\n\tValue: %d\n\tError: %s",
			key, value, err,
		)
	}

	return
}

func (myDB DB) Get(key string) (decoded interface{}, err error) {
	var ok bool
	var data []byte

	if ok, err = myDB.db.Has([]byte(key), nil); !ok {
		return
	}
	if data, err = myDB.db.Get([]byte(key), nil); err != nil {
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
