// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"
	configutils "github.com/onosproject/onos-ric-sdk-go/pkg/config/utils"

	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/pdubuilder"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	sdkSub "github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"

	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger("sb-ricapie2")

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

const serviceModelID = "e2sm_kpm-v1beta1"

// E2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type E2Session struct {
	E2SubEndpoint  string
	E2SubInstance  sdkSub.Context
	SubDelTrigger  chan bool
	E2TEndpoint    string
	RicActionID    types.RicActionID
	ReportPeriodMs uint64
	AppConfig      *app.Config
	mu             sync.RWMutex
	configEventCh  chan event.Event
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string, e2subEndpoint string, ricActionID int32, reportPeriodMs uint64) *E2Session {
	log.Info("Creating RicAPIE2Session")
	return &E2Session{
		E2SubEndpoint:  e2subEndpoint,
		E2TEndpoint:    e2tEndpoint,
		RicActionID:    types.RicActionID(ricActionID),
		ReportPeriodMs: reportPeriodMs,
	}
}

// Run starts the southbound to watch indication messages
func (s *E2Session) Run(indChan chan indication.Indication, adminSession *admin.E2AdminSession) {
	log.Info("Started KPIMON Southbound session")
	s.configEventCh = make(chan event.Event)
	go func() {
		_ = s.watchConfigChanges()
	}()
	s.SubDelTrigger = make(chan bool)
	s.manageConnections(indChan, adminSession)
}

func (s *E2Session) updateReportPeriod(event event.Event) error {
	interval, err := s.AppConfig.Get(event.Key)
	if err != nil {
		return err
	}
	value, err := configutils.ToUint64(interval.Value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.ReportPeriodMs = value
	s.mu.Unlock()
	return nil
}

func (s *E2Session) processConfigEvents() {
	for configEvent := range s.configEventCh {
		if configEvent.Key == utils.ReportPeriodConfigPath {
			log.Debug("Report Period: Config Event received:", configEvent)
			err := s.updateReportPeriod(configEvent)
			if err != nil {
				log.Error(err)
			}
			err = s.deleteSuscription()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *E2Session) watchConfigChanges() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.AppConfig.Watch(ctx, s.configEventCh)
	if err != nil {
		return err
	}
	s.processConfigEvents()
	return nil
}

// manageConnections handles connections between ONOS-KPIMON and ONOS-E2T/E2Sub.
func (s *E2Session) manageConnections(indChan chan indication.Indication, adminSession *admin.E2AdminSession) {
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
		s.manageConnection(indChan, nodeIDs)

	}
}

func (s *E2Session) manageConnection(indChan chan indication.Indication, nodeIDs []string) {
	err := s.subscribeE2T(indChan, nodeIDs)
	if err != nil {
		log.Errorf("Error happens when subscription %s", err)
	}
}

func (s *E2Session) createEventTriggerData() []byte {

	rtPeriod := periodRanges.Search(int(s.ReportPeriodMs))
	log.Infof("Received period value: %v, Set the period to: %v", s.ReportPeriodMs, rtPeriod)
	e2SmKpmEventTriggerDefinition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(int32(rtPeriod))
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
		log.Errorf("Failed to marshal event trigger definition: %v", err)
	}

	return protoBytes
}

func (s *E2Session) createSubscriptionRequest(nodeID string) (subscription.SubscriptionDetails, error) {

	return subscription.SubscriptionDetails{
		E2NodeID: subscription.E2NodeID(nodeID),
		ServiceModel: subscription.ServiceModel{
			ID: subscription.ServiceModelID(serviceModelID),
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
	}, nil
}

func (s *E2Session) subscribeE2T(indChan chan indication.Indication, nodeIDs []string) error {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)

	e2SubHost := strings.Split(s.E2SubEndpoint, ":")[0]
	e2SubPort, err := strconv.Atoi(strings.Split(s.E2SubEndpoint, ":")[1])
	if err != nil {
		log.Error("onos-e2sub's port information or endpoint information is wrong.")
		return err
	}

	clientConfig := e2client.Config{
		AppID: "onos-kpimon",
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

	subReq, err := s.createSubscriptionRequest(nodeIDs[0])
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

func (s *E2Session) deleteSuscription() error {
	err := s.E2SubInstance.Close()
	if err != nil {
		log.Error(err)
	}
	s.SubDelTrigger <- true
	return err
}
