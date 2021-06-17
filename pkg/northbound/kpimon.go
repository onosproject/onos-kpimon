// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-kpimon/pkg/utils"

	kpimonapi "github.com/onosproject/onos-api/go/onos/kpimon"
	"github.com/onosproject/onos-kpimon/pkg/store/event"
	measurementStore "github.com/onosproject/onos-kpimon/pkg/store/measurements"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"google.golang.org/grpc"
)

// NewService returns a new KPIMON interface service.
func NewService(store measurementStore.Store) service.Service {
	return &Service{
		measurementStore: store,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	measurementStore measurementStore.Store
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		measurementStore: s.measurementStore,
	}
	kpimonapi.RegisterKpimonServer(r, server)
}

// Server implements the KPIMON gRPC service for administrative facilities.
type Server struct {
	measurementStore measurementStore.Store
}

// ListMeasurements get a snapshot of measurements
func (s *Server) ListMeasurements(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	ch := make(chan measurementStore.Entry)
	measurements := make(map[string]*kpimonapi.MeasurementItems)

	go func(measurements map[string]*kpimonapi.MeasurementItems, ch chan measurementStore.Entry) {
		for entry := range ch {
			measItems := utils.ParseEntry(entry)
			cellID := entry.Key.CellIdentity.CellID
			nodeID := entry.Key.NodeID
			keyID := fmt.Sprintf("%s:%s", nodeID, cellID)
			measurements[keyID] = measItems
		}
	}(measurements, ch)

	err := s.measurementStore.Entries(ctx, ch)
	close(ch)

	if err != nil {
		return nil, err
	}

	response := &kpimonapi.GetResponse{
		Measurements: measurements,
	}

	return response, nil
}

// WatchMeasurements get measurements in a stream
func (s *Server) WatchMeasurements(request *kpimonapi.GetRequest, server kpimonapi.Kpimon_WatchMeasurementsServer) error {
	ch := make(chan event.Event)
	err := s.measurementStore.Watch(server.Context(), ch)
	if err != nil {
		return err
	}

	for e := range ch {
		measurements := make(map[string]*kpimonapi.MeasurementItems)
		measEntry := e.Value.(*measurementStore.Entry)
		// key := e.Key.(measurementStore.Key)
		cellID := measEntry.Key.CellIdentity.CellID
		nodeID := measEntry.Key.NodeID
		keyID := fmt.Sprintf("%s:%s", nodeID, cellID)

		measItems := utils.ParseEntry(*measEntry)
		measurements[keyID] = measItems

		err := server.Send(&kpimonapi.GetResponse{
			Measurements: measurements,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
