// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package subscription

import (
	"context"
	"strings"

	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"

	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"

	subutils "github.com/onosproject/onos-kpimon/pkg/utils/subscription"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	prototypes "github.com/gogo/protobuf/types"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-kpimon/pkg/rnib"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
)

var log = logging.GetLogger("e2", "subscription", "manager")

// SubManager subscription manager interface
type SubManager interface {
	Start() error
	Stop() error
}

type Manager struct {
	e2client     e2client.Client
	rnibClient   rnib.Client
	serviceModel ServiceModelOptions
}

// NewManager creates a new subscription manager
func NewManager(opts ...Option) (Manager, error) {
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}

	e2Client, err := e2client.NewClient(e2client.Config{
		AppID: "onos-kpimon",
		E2TService: e2client.ServiceConfig{
			Host: options.E2TService.Host,
			Port: options.E2TService.Port,
		},
		SubscriptionService: e2client.ServiceConfig{
			Host: options.E2SubService.Host,
			Port: options.E2SubService.Port,
		},
	})
	if err != nil {
		return Manager{}, err
	}

	rnibClient, err := rnib.NewClient()
	if err != nil {
		return Manager{}, err
	}

	return Manager{
		e2client:   e2Client,
		rnibClient: rnibClient,
		serviceModel: ServiceModelOptions{
			Name:    options.ServiceModel.Name,
			Version: options.ServiceModel.Version,
		},
	}, nil

}

// Start starts subscription manager
func (m *Manager) Start() error {
	go func() {
		err := m.watchE2Connections()
		if err != nil {
			return
		}
	}()
	return nil
}

func (m *Manager) getMeasurements(serviceModelsInfo map[string]*topoapi.ServiceModelInfo) ([]*topoapi.KPMMeasurement, error) {
	for _, sm := range serviceModelsInfo {
		smName := strings.ToLower(sm.Name)
		if smName == string(m.serviceModel.Name) {
			kpmRanFunction := &topoapi.KPMRanFunction{}
			for _, ranFunction := range sm.RanFunctions {
				if ranFunction.TypeUrl == ranFunction.GetTypeUrl() {
					err := prototypes.UnmarshalAny(ranFunction, kpmRanFunction)
					if err != nil {
						return nil, err
					}
					return kpmRanFunction.Measurements, nil
				}
			}
		}
	}
	return nil, errors.New(errors.NotFound, "cannot retrieve measurement names")

}

func (m *Manager) processIndication(ch chan indication.Indication) {
	for msg := range ch {
		log.Info("Message:", msg)
	}
}

func (m *Manager) createSubscription(ctx context.Context, nodeID topoapi.ID) error {
	log.Info("Creating subscription for E2 node with ID:", nodeID)
	aspects, err := m.rnibClient.GetE2NodeAspects(ctx, nodeID)
	if err != nil {
		log.Warn(err)
		return err
	}
	measurements, err := m.getMeasurements(aspects.ServiceModels)
	if err != nil {
		log.Warn(err)
		return err
	}

	cells, err := m.rnibClient.GetCells(ctx, nodeID)
	if err != nil {
		log.Warn(err)
		return err
	}

	eventTriggerData, err := subutils.CreateEventTriggerData(100)
	if err != nil {
		log.Warn(err)
		return err
	}

	actions, err := subutils.CreateSubscriptionActions(measurements, cells)
	if err != nil {
		log.Warn(err)
		return err
	}

	sub := subscription.SubscriptionDetails{
		E2NodeID: subscription.E2NodeID(nodeID),
		ServiceModel: subscription.ServiceModel{
			Name:    subscription.ServiceModelName(m.serviceModel.Name),
			Version: subscription.ServiceModelVersion(m.serviceModel.Version),
		},
		EventTrigger: subscription.EventTrigger{
			Payload: subscription.Payload{
				Encoding: subscription.Encoding_ENCODING_PROTO,
				Data:     eventTriggerData,
			},
		},
		Actions: actions,
	}

	ch := make(chan indication.Indication)
	_, err = m.e2client.Subscribe(ctx, sub, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	go m.processIndication(ch)

	return nil

}

func (m *Manager) watchE2Connections() error {
	ctx := context.Background()
	ch := make(chan topoapi.Event)
	err := m.rnibClient.WatchE2Connections(ctx, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	for event := range ch {
		if event.Type == topoapi.EventType_ADDED || event.Type == topoapi.EventType_NONE {
			relation := event.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			log.Info("Node ID:", e2NodeID)
			err := m.createSubscription(ctx, e2NodeID)
			if err != nil {
				log.Warn(err)
				return err
			}
		}

	}
	return nil
}

func (m *Manager) Stop() error {
	panic("implement me")
}

var _ SubManager = &Manager{}
