// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	"context"
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

// GetNumActiveUEs returns how many UEs are active
func (s Server) GetNumActiveUEs(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	return nil, nil
}
