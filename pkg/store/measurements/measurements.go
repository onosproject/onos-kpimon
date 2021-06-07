// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package measurements

import (
	"context"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// Store kpm metrics store interface
type Store interface {
	Put(ctx context.Context, key Key, value interface{}) (*Entry, error)

	// Get gets a metric store entry based on a given key
	Get(ctx context.Context, key Key) (*Entry, error)

	// Entries list all of the metric store entries
	Entries(ctx context.Context) ([]*Entry, error)

	// Keys list all of the keys
	Keys(ctx context.Context) ([]Key, error)
}

type store struct {
	measurements map[Key]*Entry
	mu           sync.RWMutex
}

// NewStore creates new store
func NewStore() Store {
	return &store{
		measurements: make(map[Key]*Entry),
	}
}

func (s *store) Entries(ctx context.Context) ([]*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := make([]*Entry, 0)
	for _, entry := range s.measurements {
		entries = append(entries, entry)
	}

	return entries, nil
}

// Keys list of measurement keys
func (s *store) Keys(ctx context.Context) ([]Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]Key, 0)
	for k := range s.measurements {
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *store) Put(ctx context.Context, key Key, value interface{}) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := &Entry{
		Value: value,
	}
	s.measurements[key] = entry
	return entry, nil

}

func (s *store) Get(ctx context.Context, key Key) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.measurements[key]; ok {
		return v, nil
	}
	return nil, errors.New(errors.NotFound, "the measurement entry does not exist")
}

// Entry metric store entry
type Entry struct {
	Value interface{}
}

// NewKey creates a new measurements map key
func NewKey(CellID CellIdentity) Key {
	return Key{
		CellIdentity: CellID,
	}
}

// Key is the key of monitoring result metric store
type Key struct {
	CellIdentity CellIdentity
}

var _ Store = &store{}
