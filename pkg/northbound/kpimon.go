// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	"context"

	"github.com/onosproject/onos-kpimon/pkg/store/event"

	"github.com/onosproject/onos-kpimon/pkg/store/measurements"

	kpimonapi "github.com/onosproject/onos-api/go/onos/kpimon"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"google.golang.org/grpc"
)

// NewService returns a new KPIMON interface service.
func NewService(store measurements.Store) service.Service {
	return &Service{
		measurementStore: store,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	measurementStore measurements.Store
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
	measurementStore measurements.Store
}

// GetMetricTypes returns all metric types - for CLI
func (s Server) GetMetricTypes(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	// ignore ID here since it will return results for all keys
	/*attr := make(map[string]string)


	s.monitor.GetKpiMonMutex().RLock()
	for key := range s.monitor.GetKpiMonResults() {
		attr[key.Metric] = "0"
	}
	s.monitor.GetKpiMonMutex().RUnlock()*/
	ch := make(chan event.Event)
	err := s.measurementStore.Watch(ctx, ch)
	if err != nil {
		return nil, err
	}

	response := &kpimonapi.GetResponse{
		Object: &kpimonapi.Object{
			Id:       "all",
			Revision: 0,
			//Attributes: attr,
		},
	}

	return response, nil
}

// GetMetrics returns all KPI monitoring results - for CLI
func (s Server) GetMetrics(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	// ignore ID here since it will return results for all keys
	attr := make(map[string]string)
	/*s.monitor.GetKpiMonMutex().Lock()
	for key, value := range s.monitor.GetKpiMonResults() {
		attr[fmt.Sprintf("%s:%s:%s:%s:%d", key.CellIdentity.CellID, key.CellIdentity.PlmnID, key.CellIdentity.ECI, key.Metric, key.Timestamp)] = value.Value
	}
	s.monitor.GetKpiMonMutex().Unlock()*/

	response := &kpimonapi.GetResponse{
		Object: &kpimonapi.Object{
			Id:         "all",
			Revision:   0,
			Attributes: attr,
		},
	}

	return response, nil
}
