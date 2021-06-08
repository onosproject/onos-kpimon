// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package subscription

import (
	"context"
	"strings"
	"time"

	"github.com/onosproject/onos-kpimon/pkg/store/actions"

	"github.com/cenkalti/backoff/v4"

	"github.com/onosproject/onos-kpimon/pkg/utils"

	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"

	"github.com/onosproject/onos-kpimon/pkg/monitoring"

	"github.com/onosproject/onos-kpimon/pkg/broker"

	"github.com/onosproject/onos-ric-sdk-go/pkg/app"

	appConfig "github.com/onosproject/onos-kpimon/pkg/config"

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

const (
	kpmServiceModelOID = "1.3.6.1.4.1.53148.1.2.2.2"
)

const (
	backoffInterval = 10 * time.Millisecond
	maxBackoffTime  = 5 * time.Second
)

func newExpBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = backoffInterval
	// MaxInterval caps the RetryInterval
	b.MaxInterval = maxBackoffTime
	// Never stops retrying
	b.MaxElapsedTime = 0
	return b
}

// SubManager subscription manager interface
type SubManager interface {
	Start() error
	Stop() error
}

// Manager subscription manager
type Manager struct {
	e2client     e2client.Client
	rnibClient   rnib.Client
	serviceModel ServiceModelOptions
	appConfig    *appConfig.AppConfig
	streams      broker.Broker
	monitor      *monitoring.Monitor
	actionStore  actions.Store
}

// NewManager creates a new subscription manager
func NewManager(opts ...Option) (Manager, error) {
	options := Options{}

	for _, opt := range opts {
		opt.apply(&options)
	}

	e2Client, err := e2client.NewClient(e2client.Config{
		AppID: app.ID(options.App.AppID),
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
		appConfig:   options.App.AppConfig,
		streams:     options.App.Broker,
		monitor:     options.App.Monitor,
		actionStore: options.App.Actions,
	}, nil

}

// Start starts subscription manager
func (m *Manager) Start() error {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := m.watchE2Connections(ctx)
		if err != nil {
			return
		}
	}()

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := m.watchConfigChanges(ctx)
		if err != nil {
			return
		}
	}()
	return nil
}

func (m *Manager) watchConfigChanges(ctx context.Context) error {
	ch := make(chan event.Event)
	err := m.appConfig.Watch(ctx, ch)
	if err != nil {
		return err
	}

	// Deletes all of subscriptions
	for configEvent := range ch {
		if configEvent.Key == utils.ReportPeriodConfigPath {
			subIDs := m.streams.SubIDs()
			for _, subID := range subIDs {
				_, err := m.streams.CloseStream(subID)
				if err != nil {
					log.Warn(err)
					return err
				}
			}
		}

	}
	// Gets all of connected E2 nodes and creates new subscriptions based on new report interval
	e2NodeIDs, err := m.rnibClient.E2NodeIDs(ctx)
	if err != nil {
		log.Warn(err)
		return err
	}

	for _, e2NodeID := range e2NodeIDs {
		go func(e2NodeID topoapi.ID) {
			err := m.newSubscription(ctx, e2NodeID)
			if err != nil {
				log.Warn(err)
			}
		}(e2NodeID)
	}

	return nil

}

func (m *Manager) getMeasurements(serviceModelsInfo map[string]*topoapi.ServiceModelInfo) ([]*topoapi.KPMMeasurement, error) {
	for _, sm := range serviceModelsInfo {
		smName := strings.ToLower(sm.Name)
		if smName == string(m.serviceModel.Name) && sm.OID == kpmServiceModelOID {
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

func (m *Manager) sendIndicationOnStream(streamID broker.StreamID, ch chan indication.Indication) {
	streamWriter, err := m.streams.GetStream(streamID)
	if err != nil {
		return
	}

	for msg := range ch {
		err := streamWriter.Send(msg)
		if err != nil {
			log.Warn(err)
			return
		}
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

	reportPeriod, err := m.appConfig.GetReportPeriod()
	if err != nil {
		log.Warn(err)
		return err
	}
	eventTriggerData, err := subutils.CreateEventTriggerData(uint32(reportPeriod))
	if err != nil {
		log.Warn(err)
		return err
	}

	granularityPeriod, err := m.appConfig.GetGranularityPeriod()
	if err != nil {
		log.Warn(err)
		return err
	}

	actions, err := m.createSubscriptionActions(ctx, measurements, cells, uint32(granularityPeriod))
	if err != nil {
		log.Warn(err)
		return err
	}

	subRequest := subscription.SubscriptionDetails{
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
	sub, err := m.e2client.Subscribe(ctx, subRequest, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	stream, err := m.streams.OpenStream(sub)
	if err != nil {
		return err
	}

	go m.sendIndicationOnStream(stream.StreamID(), ch)
	go func() {
		err = m.monitor.Start(ctx, sub, measurements)
		if err != nil {
			log.Warn(err)
		}

	}()

	return nil

}

func (m *Manager) newSubscription(ctx context.Context, e2NodeID topoapi.ID) error {
	// TODO revisit this after migrating to use new E2 sdk
	count := 0
	notifier := func(err error, t time.Duration) {
		count++
		log.Infof("Retrying, failed to create subscription for E2 node with ID %s due to %s", e2NodeID, err)
	}

	err := backoff.RetryNotify(func() error {
		err := m.createSubscription(ctx, e2NodeID)
		return err
	}, newExpBackoff(), notifier)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) watchE2Connections(ctx context.Context) error {
	ch := make(chan topoapi.Event)
	err := m.rnibClient.WatchE2Connections(ctx, ch)
	if err != nil {
		log.Warn(err)
		return err
	}

	// creates a new subscription whenever there is a new E2 node connected and supports KPM service model
	for topoEvent := range ch {
		if topoEvent.Type == topoapi.EventType_ADDED || topoEvent.Type == topoapi.EventType_NONE {
			relation := topoEvent.Object.Obj.(*topoapi.Object_Relation)
			e2NodeID := relation.Relation.TgtEntityID
			go func() {
				err := m.newSubscription(ctx, e2NodeID)
				if err != nil {
					log.Warn(err)
				}
			}()
		}

	}
	return nil
}

// Stop stops the subscription manager
func (m *Manager) Stop() error {
	panic("implement me")
}

var _ SubManager = &Manager{}
