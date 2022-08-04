package badger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	badgerDB "github.com/dgraph-io/badger/v3"
)

func Open(path string) (*badgerDB.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
	opts := badgerDB.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path
	opts.SyncWrites = false
	opts.ValueThreshold = 256
	opts.CompactL0OnClose = true

	// using memory
	// opts := badgerDB.DefaultOptions(path).WithInMemory(true)

	db, err := badgerDB.Open(opts)
	if err != nil {
		log.Println("badger open failed", "path", path, "err", err)
		return nil, err
	}
	return db, nil
}

func Close() {
	err := badgerDB.Close()
	if err == nil {
		log.Println("database closed", "err", err)
	} else {
		log.Println("failed to close database", "err", err)
	}
}

func Set(key []byte, value []byte) {
	wb := badgerDB.NewWriteBatch()
	defer wb.Cancel()
	err := wb.SetEntry(badgerDB.NewEntry(key, value).WithMeta(0))
	if err != nil {
		log.Println("Failed to write data to cache.", "key", string(key), "value", string(value), "err", err)
	}
	err = wb.Flush()
	if err != nil {
		log.Println("Failed to flush data to cache.", "key", string(key), "value", string(value), "err", err)
	}
}

func SetWithTTL(key []byte, value []byte, ttl int64) {
	wb := badgerDB.NewWriteBatch()
	defer wb.Cancel()
	err := wb.SetEntry(badgerDB.NewEntry(key, value).WithMeta(0).WithTTL(time.Duration(ttl * time.Second.Nanoseconds())))
	if err != nil {
		log.Println("Failed to write data to cache.", "key", string(key), "value", string(value), "err", err)
	}
	err = wb.Flush()
	if err != nil {
		log.Println("Failed to flush data to cache.", "key", string(key), "value", string(value), "err", err)
	}
}

func Get(key []byte) string {
	var ival []byte
	err := badgerDB.View(func(txn *badgerDB.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		ival, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		log.Println("Failed to read data from the cache.", "key", string(key), "error", err)
	}
	return string(ival)
}

func Has(key []byte) (bool, error) {
	var exist bool = false
	err := badgerDB.View(func(txn *badgerDB.Txn) error {
		_, err := txn.Get(key)
		if err != nil {
			return err
		} else {
			exist = true
		}
		return err
	})
	// align with leveldb, if the key doesn't exist, leveldb returns nil
	if strings.HasSuffix(err.Error(), "not found") {
		err = nil
	}
	return exist, err
}

func Delete(key []byte) error {
	wb := badgerDB.NewWriteBatch()
	defer wb.Cancel()
	return wb.Delete(key)
}

func IteratorKeysAndValues() {

	err := badgerDB.View(func(txn *badgerDB.Txn) error {
		opts := badgerDB.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Failed to iterator keys and values from the cache.", "error", err)
	}
}

func IteratorKeys() {
	err := badgerDB.View(func(txn *badgerDB.Txn) error {
		opts := badgerDB.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			fmt.Printf("key=%s\n", k)
		}
		return nil
	})

	if err != nil {
		log.Println("Failed to iterator keys from the cache.", "error", err)
	}
}

func SeekWithPrefix(prefixStr string) {
	err := badgerDB.View(func(txn *badgerDB.Txn) error {
		it := txn.NewIterator(badgerDB.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(prefixStr)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Failed to seek prefix from the cache.", "prefix", prefixStr, "error", err)
	}
}
