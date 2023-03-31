package store

import (
	badger2 "github.com/dgraph-io/badger/v2"
	badger "github.com/ipfs/go-ds-badger2"
)

// DefaultBadgerOptions creates the default options for badger.
// For our purposes, we don't want the store to perform any garbage collection or
// expire newly added keys after a certain period, because:
// 1. the data in the store will be light.
// 2. we want to keep the data, i.e. confirms, for the longest time possible to be able
// to retrieve them if needed.
func DefaultBadgerOptions(path string) *badger.Options {
	return &badger.Options{
		GcDiscardRatio: 0,
		GcInterval:     0,
		GcSleep:        0,
		TTL:            0,
		Options:        badger2.DefaultOptions(path),
	}
}
