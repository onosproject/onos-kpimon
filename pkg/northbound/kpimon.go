// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package northbound

import (
	"context"

	prototypes "github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-kpimon/pkg/store/event"
	measurementStore "github.com/onosproject/onos-kpimon/pkg/store/measurements"
	"github.com/onosproject/onos-lib-go/pkg/logging"

	kpimonapi "github.com/onosproject/onos-api/go/onos/kpimon"
	"github.com/onosproject/onos-lib-go/pkg/logging/service"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound")

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

// GetMeasurementTypes get measurement types
func (s *Server) GetMeasurementTypes(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	panic("implement me")
}

// GetMeasurement get a snapshot of measurements
func (s *Server) GetMeasurement(ctx context.Context, request *kpimonapi.GetRequest) (*kpimonapi.GetResponse, error) {
	panic("implement me")
}

// GetMeasurements get measurements in a stream
func (s *Server) GetMeasurements(request *kpimonapi.GetRequest, server kpimonapi.Kpimon_GetMeasurementsServer) error {
	ch := make(chan event.Event)
	err := s.measurementStore.Watch(server.Context(), ch)
	if err != nil {
		return err
	}

	for e := range ch {
		measurements := make(map[string]*kpimonapi.MeasurementItems)
		measEntry := e.Value.(*measurementStore.Entry)
		key := e.Key.(measurementStore.Key)
		measEntryItems := measEntry.Value.([]measurementStore.MeasurementItem)
		measItem := &kpimonapi.MeasurementItem{}
		measItems := &kpimonapi.MeasurementItems{}
		for _, entryItem := range measEntryItems {
			measItem.MeasurementRecords = make([]*kpimonapi.MeasurementRecord, 0)
			for _, record := range entryItem.MeasurementRecords {
				var value *prototypes.Any
				switch val := record.MeasurementValue.(type) {
				case int64:
					intValue := &kpimonapi.IntegerValue{Value: val}
					value, err = prototypes.MarshalAny(intValue)
					if err != nil {
						log.Warn(err)
						continue
					}

				case float64:
					realValue := &kpimonapi.RealValue{
						Value: val,
					}
					value, err = prototypes.MarshalAny(realValue)
					if err != nil {
						log.Warn(err)
						continue
					}
				case int32:
					noValue := &kpimonapi.NoValue{
						Value: val,
					}
					value, err = prototypes.MarshalAny(noValue)
					if err != nil {
						log.Warn(err)
						continue
					}

				}

				measRecord := &kpimonapi.MeasurementRecord{
					MeasurementName:  record.MeasurementName,
					Timestamp:        record.Timestamp,
					MeasurementValue: value,
				}
				measItem.MeasurementRecords = append(measItem.MeasurementRecords, measRecord)
			}

			measItems.MeasurementItems = append(measItems.MeasurementItems, measItem)

		}
		measurements[key.CellIdentity.CellID] = measItems
		err := server.Send(&kpimonapi.GetResponse{
			Measurements: measurements,
		})
		if err != nil {
			return err
		}
	}
	return nil

}
