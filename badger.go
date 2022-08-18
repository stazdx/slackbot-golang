package main

import (
	"log"
	"os"

	badger "github.com/dgraph-io/badger/v3"
)

func Open(path string) (*badger.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path
	opts.SyncWrites = false
	opts.ValueThreshold = 256
	opts.CompactL0OnClose = true

	// using memory
	// opts := badger.DefaultOptions(path).WithInMemory(true)

	db, err := badger.Open(opts)
	if err != nil {
		log.Println("badger open failed", "path", path, "err", err)
		return nil, err
	}
	return db, nil
}
