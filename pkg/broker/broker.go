// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package broker

import (
	"io"
	"sync"

	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2sub "github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"
)

var log = logging.GetLogger("broker")

// NewBroker creates a new subscription stream broker
func NewBroker() Broker {
	return &streamBroker{
		subs:    make(map[subscription.ID]Stream),
		streams: make(map[StreamID]Stream),
	}
}

// Broker is a subscription stream broker
// The Broker is responsible for managing Streams for propagating indications from the southbound API
// to the northbound API.
type Broker interface {
	io.Closer

	// OpenStream opens a subscription Stream
	// If a stream already exists for the subscription, the existing stream will be returned.
	// If no stream exists, a new stream will be allocated with a unique StreamID.
	OpenStream(e2SubCtx e2sub.Context) (StreamReader, error)

	// CloseStream closes a subscription Stream
	// The associated Stream will be closed gracefully: the reader will continue receiving pending indications
	// until the buffer is empty.
	CloseStream(id subscription.ID) (StreamReader, error)

	// GetStream gets a write stream by its StreamID
	// If no Stream exists for the given StreamID, a NotFound error will be returned.
	GetStream(id StreamID) (StreamWriter, error)

	// SubIDs get all of subscription IDs
	SubIDs() []subscription.ID
}

type streamBroker struct {
	subs     map[subscription.ID]Stream
	streams  map[StreamID]Stream
	streamID StreamID
	mu       sync.RWMutex
}

func (b *streamBroker) SubIDs() []subscription.ID {
	b.mu.Lock()
	defer b.mu.Unlock()
	subIDs := make([]subscription.ID, len(b.subs))
	for subID := range b.subs {
		subIDs = append(subIDs, subID)
	}
	return subIDs
}

func (b *streamBroker) OpenStream(e2sub e2sub.Context) (StreamReader, error) {
	b.mu.RLock()
	stream, ok := b.subs[e2sub.ID()]
	b.mu.RUnlock()
	if ok {
		return stream, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	stream, ok = b.subs[e2sub.ID()]
	if ok {
		return stream, nil
	}

	b.streamID++
	streamID := b.streamID
	stream = newBufferedStream(e2sub.ID(), streamID, e2sub)
	b.subs[e2sub.ID()] = stream
	b.streams[streamID] = stream
	log.Infof("Opened new stream %d for subscription '%s'", streamID, e2sub.ID())
	return stream, nil
}

func (b *streamBroker) CloseStream(id subscription.ID) (StreamReader, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	stream, ok := b.subs[id]
	if !ok {
		return nil, errors.NewNotFound("subscription '%s' not found", id)
	}
	delete(b.subs, stream.SubscriptionID())
	delete(b.streams, stream.StreamID())
	err := stream.SubContext().Close()
	if err != nil {
		return nil, err
	}
	log.Infof("Closed stream %d for subscription '%s'", stream.StreamID(), id)
	return stream, stream.Close()
}

func (b *streamBroker) GetStream(id StreamID) (StreamWriter, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	stream, ok := b.streams[id]
	if !ok {
		return nil, errors.NewNotFound("stream %d not found", id)
	}
	return stream, nil
}

func (b *streamBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var err error
	for _, stream := range b.streams {
		if e := stream.Close(); e != nil {
			err = e
		}
	}
	return err
}

var _ Broker = &streamBroker{}
