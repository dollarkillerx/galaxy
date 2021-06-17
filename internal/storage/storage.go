package storage

import (
	"github.com/dgraph-io/badger/v3"

	"log"
)

type storage struct {
	db *badger.DB
}

var Storage *storage

func init() {
	open, err := badger.Open(badger.DefaultOptions("./galaxy_data"))
	if err != nil {
		log.Fatalln(err)
	}

	Storage = &storage{db: open}
}
