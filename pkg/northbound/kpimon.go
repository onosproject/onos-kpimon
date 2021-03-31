// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	"context"
	"fmt"
	kpimonapi "github.com/onosproject/onos-api/go/onos/kpimon"
	"github.com/onosproject/onos-kpimon/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"google.golang.org/grpc"
)

// NewService returns a new KPIMON interface service.
func NewService(ctrl controller.KpiMonController) service.Service {
	return &Service{
		Ctrl: ctrl,
	}
}

// Service is a service implementation for administration.
type Service struct {
	service.Service
	Ctrl controller.KpiMonController
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		Ctrl: s.Ctrl,
	}
	kpimonapi.RegisterKpimonServer(r, server)
}

// Server implements the KPIMON gRPC service for administrative facilities.
type Server struct {
	Ctrl controller.KpiMonController
}

// GetMetricTypes returns all metric types - for CLI
func (s Server) GetMetricTypes(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	// ignore ID here since it will return results for all keys
	attr := make(map[string]string)

	s.Ctrl.GetKpiMonMutex().RLock()
	for key := range s.Ctrl.GetKpiMonResults() {
		attr[key.Metric] = "0"
	}
	s.Ctrl.GetKpiMonMutex().RUnlock()

	response := &kpimonapi.GetResponse{
		Object: &kpimonapi.Object{
			Id:         "all",
			Revision:   0,
			Attributes: attr,
		},
	}

	return response, nil
}

// GetMetrics returns all KPI monitoring results - for CLI
func (s Server) GetMetrics(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	// ignore ID here since it will return results for all keys
	attr := make(map[string]string)

	s.Ctrl.GetKpiMonMutex().Lock()
	for key, value := range s.Ctrl.GetKpiMonResults() {
		attr[fmt.Sprintf("%s:%s:%s", key.CellIdentity.PlmnID, key.CellIdentity.ECI, key.Metric)] = value.Value
	}
	s.Ctrl.GetKpiMonMutex().Unlock()

	response := &kpimonapi.GetResponse{
		Object: &kpimonapi.Object{
			Id:         "all",
			Revision:   0,
			Attributes: attr,
		},
	}

	return response, nil
}
