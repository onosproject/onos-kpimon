// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"
	configutils "github.com/onosproject/onos-ric-sdk-go/pkg/config/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	sdkSub "github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"
	"sync"
)

var log = logging.GetLogger("southbound", "ricapie2")

// NewE2Session generates a new E2Session
func NewE2Session(e2tEndpoint string, e2subEndpoint string, ricActionID int32, reportPeriodMs uint64, smName string, smVersion string, kpiMonMetricMap map[int]string) E2Session {
	var e2Session E2Session
	if smVersion == "v1" {
		e2Session = newV1E2Session(e2tEndpoint, e2subEndpoint, ricActionID, reportPeriodMs, smName, smVersion, kpiMonMetricMap)
	} else if smVersion == "v2" {
		e2Session = newV2E2Session(e2tEndpoint, e2subEndpoint, ricActionID, reportPeriodMs, smName, smVersion, kpiMonMetricMap)
	} else {
		// It shouldn't be hit
		log.Fatal("The received service model version %s is not valid - it must be v1 or v2", smVersion)
	}
	return e2Session
}

// E2Session is an interface of E2 Session
type E2Session interface {
	Run(chan indication.Indication, admin.E2AdminSession)
	SetReportPeriodMs(uint64)
	SetGranularityMs(uint64)
	SetAppConfig(*app.Config)
	GetKpiMonMetricMap() map[int]string
	updateReportPeriod(event.Event) error
	processConfigEvents()
	watchConfigChanges() error
	deleteE2Subscription(string) error
	deleteE2Subscriptions() error
}

// AbstractE2Session is an abstract struct of E2 session
type AbstractE2Session struct {
	E2Session
	E2SubEndpoint   string
	E2SubInstances  map[string]sdkSub.Context
	SubDelTriggers  map[string]chan bool
	E2TEndpoint     string
	RicActionID     types.RicActionID
	ReportPeriodMs  uint64
	GranularityMs   uint64
	AppConfig       *app.Config
	EventMutex      sync.RWMutex
	ConfigEventCh   chan event.Event
	SMName          string
	SMVersion       string
	KpiMonMetricMap map[int]string
}

// SetReportPeriodMs changes the ReportPeriodMS
func (s *AbstractE2Session) SetReportPeriodMs(period uint64) {
	s.ReportPeriodMs = period
}

// SetGranularityMs changes the GranularityPeriodMS
func (s *AbstractE2Session) SetGranularityMs(granularity uint64) {
	s.GranularityMs = granularity
}

// SetAppConfig sets AppConfig
func (s *AbstractE2Session) SetAppConfig(appConfig *app.Config) {
	s.AppConfig = appConfig
}

func (s *AbstractE2Session) updateReportPeriod(event event.Event) error {
	interval, err := s.AppConfig.Get(event.Key)
	if err != nil {
		return err
	}

	value, err := configutils.ToUint64(interval.Value)
	if err != nil {
		return err
	}

	s.EventMutex.Lock()
	s.ReportPeriodMs = value
	s.EventMutex.Unlock()

	return nil
}

func (s *AbstractE2Session) processConfigEvents() {
	for configEvent := range s.ConfigEventCh {
		if configEvent.Key == utils.ReportPeriodConfigPath {
			log.Infof("Report Period: Config Event received: %v", configEvent)
			err := s.updateReportPeriod(configEvent)
			if err != nil {
				log.Error(err)
			}
			err = s.deleteE2Subscriptions()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *AbstractE2Session) watchConfigChanges() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.AppConfig.Watch(ctx, s.ConfigEventCh)
	if err != nil {
		return err
	}
	s.processConfigEvents()
	return nil
}

func (s *AbstractE2Session) deleteE2Subscriptions() error {
	for k := range s.E2SubInstances {
		err := s.deleteE2Subscription(k)
		if err != nil {
			log.Errorf("Failed to delete E2 subscription: %v", err)
			return err
		}
		s.SubDelTriggers[k] <- true
	}
	return nil
}

func (s *AbstractE2Session) deleteE2Subscription(e2NodeID string) error {
	log.Infof("Start deleting subscription - E2NodeID: %v", e2NodeID)
	err := s.E2SubInstances[e2NodeID].Close()
	return err
}

// GetKpiMonMetricMap returns the KpiMonMetricMap
func (s *V1E2Session) GetKpiMonMetricMap() map[int]string {
	return s.KpiMonMetricMap
}
