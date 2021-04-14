// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/pdubuilder"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

var periodRanges = utils.PeriodRanges{
	{Min: 0, Max: 10, Value: 0},
	{Min: 11, Max: 20, Value: 1},
	{Min: 21, Max: 32, Value: 2},
	{Min: 33, Max: 40, Value: 3},
	{Min: 41, Max: 60, Value: 4},
	{Min: 61, Max: 64, Value: 5},
	{Min: 65, Max: 70, Value: 6},
	{Min: 71, Max: 80, Value: 7},
	{Min: 81, Max: 128, Value: 8},
	{Min: 129, Max: 160, Value: 9},
	{Min: 161, Max: 256, Value: 10},
	{Min: 257, Max: 320, Value: 11},
	{Min: 321, Max: 512, Value: 12},
	{Min: 513, Max: 640, Value: 13},
	{Min: 641, Max: 1024, Value: 14},
	{Min: 1025, Max: 1280, Value: 15},
	{Min: 1281, Max: 2048, Value: 16},
	{Min: 2049, Max: 2560, Value: 17},
	{Min: 2561, Max: 5120, Value: 18},
	{Min: 5121, Max: math.MaxInt64, Value: 19},
}

func newV1E2Session(e2tEndpoint string, e2subEndpoint string, ricActionID int32, reportPeriodMs uint64, smName string, smVersion string, kpimonMetricMap map[int]string) *V1E2Session {
	log.Info("Creating RICAPI E2Session for KPM v1.0")
	kpimonMetricMap[1] = "numActiveUEs"
	return &V1E2Session{
		AbstractE2Session: &AbstractE2Session{
			E2SubEndpoint:   e2subEndpoint,
			E2TEndpoint:     e2tEndpoint,
			RicActionID:     types.RicActionID(ricActionID),
			ReportPeriodMs:  reportPeriodMs,
			SMName:          smName,
			SMVersion:       smVersion,
			KpiMonMetricMap: kpimonMetricMap,
		},
	}
}

// V1E2Session is an E2 session for KPM v1.0
type V1E2Session struct {
	*AbstractE2Session
}

// Run starts E2 session
func (s *V1E2Session) Run(indChan chan indication.Indication, adminSession admin.E2AdminSession) {
	log.Info("Started KPIMON Southbound session")
	s.ConfigEventCh = make(chan event.Event)
	go func() {
		_ = s.watchConfigChanges()
	}()
	s.SubDelTrigger = make(chan bool)
	s.manageConnections(indChan, adminSession)
}

func (s *V1E2Session) manageConnections(indChan chan indication.Indication, adminSession admin.E2AdminSession) {
	for {
		nodeIDs, err := adminSession.GetListE2NodeIDs()
		if err != nil {
			log.Errorf("Cannot get NodeIDs through Admin API: %s", err)
			continue
		} else if len(nodeIDs) == 0 {
			log.Warn("CU-CP is not running - wait until CU-CP is ready")
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		log.Infof("Received E2Nodes: %v", nodeIDs)
		var wg sync.WaitGroup
		for _, id := range nodeIDs {
			wg.Add(1)
			go func(id string, wg *sync.WaitGroup) {
				defer wg.Done()
				for {
					s.manageConnection(indChan, id)
				}
			}(id, &wg)
		}
		wg.Wait()
	}
}

func (s *V1E2Session) manageConnection(indChan chan indication.Indication, nodeID string) {
	err := s.createE2Subscription(indChan, nodeID)
	if err != nil {
		log.Errorf("Error happens when creating E2 subscription - %s", err)
	}
}

func (s *V1E2Session) createSubscriptionRequest(nodeID string) (subscription.SubscriptionDetails, error) {
	sub := subscription.SubscriptionDetails{
		E2NodeID: subscription.E2NodeID(nodeID),
		ServiceModel: subscription.ServiceModel{
			Name:    subscription.ServiceModelName(s.SMName),
			Version: subscription.ServiceModelVersion(s.SMVersion),
		},
		EventTrigger: subscription.EventTrigger{
			Payload: subscription.Payload{
				Encoding: subscription.Encoding_ENCODING_PROTO,
				Data:     s.createEventTriggerData(),
			},
		},
		Actions: []subscription.Action{
			{
				ID:   int32(s.RicActionID),
				Type: subscription.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &subscription.SubsequentAction{
					Type:       subscription.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: subscription.TimeToWait_TIME_TO_WAIT_ZERO,
				},
			},
		},
	}
	log.Debugf("sub: %v", sub)

	return sub, nil
}

func (s *V1E2Session) createE2Subscription(indChan chan indication.Indication, nodeID string) error {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)

	e2SubHost := strings.Split(s.E2SubEndpoint, ":")[0]
	e2SubPort, err := strconv.Atoi(strings.Split(s.E2SubEndpoint, ":")[1])
	if err != nil {
		log.Error("onos-e2sub's port information or endpoint information is wrong.")
		return err
	}
	e2tHost := strings.Split(s.E2TEndpoint, ":")[0]
	e2tPort, err := strconv.Atoi(strings.Split(s.E2TEndpoint, ":")[1])
	if err != nil {
		log.Error("onos-e2t's port information or endpoint information is wrong.")
		return err
	}

	clientConfig := e2client.Config{
		AppID: "onos-kpimon-v1",
		E2TService: e2client.ServiceConfig{
			Host: e2tHost,
			Port: e2tPort,
		},
		SubscriptionService: e2client.ServiceConfig{
			Host: e2SubHost,
			Port: e2SubPort,
		},
	}

	client, err := e2client.NewClient(clientConfig)
	if err != nil {
		log.Error("Can't open E2Client.")
		return err
	}

	ch := make(chan indication.Indication)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subReq, err := s.createSubscriptionRequest(nodeID)
	if err != nil {
		log.Error("Can't create SubsdcriptionRequest message")
		return err
	}

	s.E2SubInstance, err = client.Subscribe(ctx, subReq, ch)
	if err != nil {
		log.Error("Can't send SubscriptionRequest message")
		return err
	}

	log.Infof("Start forwarding Indication message to KPIMON controller")
	for {
		select {
		case indMsg := <-ch:
			indChan <- indMsg
		case trigger := <-s.SubDelTrigger:
			if trigger {
				log.Info("Reset indChan to close subscription")
				return nil
			}
		}
	}
}

func (s *V1E2Session) createEventTriggerData() []byte {
	rtPeriod := s.getReportPeriodFromAdmin()
	log.Infof("Received period value: %v, set the period to: %v", s.ReportPeriodMs, rtPeriod)
	e2SmKpmEventTriggerDefinition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(rtPeriod)
	if err != nil {
		log.Errorf("Failed to create event trigger definition data: %v", err)
		return []byte{}
	}

	err = e2SmKpmEventTriggerDefinition.Validate()
	if err != nil {
		log.Errorf("Failed to validate the event trigger definition: %v", err)
		return []byte{}
	}

	protoBytes, err := proto.Marshal(e2SmKpmEventTriggerDefinition)
	if err != nil {
		log.Errorf("Failed to marshal event trigger definition %v", err)
		return []byte{}
	}

	return protoBytes
}

func (s *V1E2Session) getReportPeriodFromAdmin() int32 {
	rtPeriod := periodRanges.Search(int(s.ReportPeriodMs))
	return int32(rtPeriod)
}
