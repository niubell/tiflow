// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package message

import (
	"fmt"

	"github.com/pingcap/errors"
	"github.com/pingcap/ticdc/cdc/sorter/encoding"
	"github.com/pingcap/ticdc/pkg/db"
	"golang.org/x/sync/semaphore"
)

// Task is a db actor task. It carries write and read request.
type Task struct {
	UID     uint32
	TableID uint64

	// encoded key -> serde.marshal(event)
	// If a value is empty, it deletes the key/value entry in db.
	Events map[Key][]byte
	Iter   *IterParam

	// For clean-up table task.
	Cleanup            bool
	CleanupRatelimited bool
}

type IterParam struct {
	UID     uint32
	TableID uint64

	ResolvedTs uint64
	Range      [2][]byte
	// Must be buffered channel to avoid blocking.
	IterCh chan *LimitedIterator `json:"-"` // Make Task JSON printable.
}

// Key is the key that is written to db.
type Key string

// String returns a pretty printed string.
func (k Key) String() string {
	uid, tableID, startTs, CRTs := encoding.DecodeKey([]byte(k))
	return fmt.Sprintf(
		"uid: %d, tableID: %d, startTs: %d, CRTs: %d",
		uid, tableID, startTs, CRTs)
}

// LimitedIterator is a wrapper of db.Snapshot that has a sema to limit
// the total number of alive snapshots.
type LimitedIterator struct {
	db.Iterator
	Sema       *semaphore.Weighted
	ResolvedTs uint64
}

// Release resources of the snapshot.
func (s *LimitedIterator) Release() error {
	s.Sema.Release(1)
	return errors.Trace(s.Iterator.Release())
}

// NewCleanupTask returns a clean up task to clean up table data.
func NewCleanupTask(uid uint32, tableID uint64) Task {
	return Task{
		TableID: tableID,
		UID:     uid,
		Cleanup: true,
	}
}
