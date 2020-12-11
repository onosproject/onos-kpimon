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
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
	"strconv"
	"strings"
	"time"
)

var log = logging.GetLogger("sb-ricapie2")

const serviceModelID = "e2sm_kpm-v1beta1"

// E2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type E2Session struct {
	E2SubEndpoint  string
	E2TEndpoint    string
	RicActionID    types.RicActionID
	ReportPeriodMs uint64
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
	s.manageConnections(indChan, adminSession)
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

func (s *E2Session) getReportPeriod() int32 {
	if s.ReportPeriodMs <= 10 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS10 period", s.ReportPeriodMs)
		return 0
	} else if s.ReportPeriodMs <= 20 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS20 period", s.ReportPeriodMs)
		return 1
	} else if s.ReportPeriodMs <= 32 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS32 period", s.ReportPeriodMs)
		return 2
	} else if s.ReportPeriodMs <= 40 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS40 period", s.ReportPeriodMs)
		return 3
	} else if s.ReportPeriodMs <= 60 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS60 period", s.ReportPeriodMs)
		return 4
	} else if s.ReportPeriodMs <= 64 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS64 period", s.ReportPeriodMs)
		return 5
	} else if s.ReportPeriodMs <= 70 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS70 period", s.ReportPeriodMs)
		return 6
	} else if s.ReportPeriodMs <= 80 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS80 period", s.ReportPeriodMs)
		return 7
	} else if s.ReportPeriodMs <= 128 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS128 period", s.ReportPeriodMs)
		return 8
	} else if s.ReportPeriodMs <= 160 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS160 period", s.ReportPeriodMs)
		return 9
	} else if s.ReportPeriodMs <= 256 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS256 period", s.ReportPeriodMs)
		return 10
	} else if s.ReportPeriodMs <= 320 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS320 period", s.ReportPeriodMs)
		return 11
	} else if s.ReportPeriodMs <= 512 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS512 period", s.ReportPeriodMs)
		return 12
	} else if s.ReportPeriodMs <= 640 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS640 period", s.ReportPeriodMs)
		return 13
	} else if s.ReportPeriodMs <= 1024 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS1024 period", s.ReportPeriodMs)
		return 14
	} else if s.ReportPeriodMs <= 1280 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS1280 period", s.ReportPeriodMs)
		return 15
	} else if s.ReportPeriodMs <= 2048 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS2048 period", s.ReportPeriodMs)
		return 16
	} else if s.ReportPeriodMs <= 2560 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS2560 period", s.ReportPeriodMs)
		return 17
	} else if s.ReportPeriodMs <= 5120 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS5120 period", s.ReportPeriodMs)
		return 18
	} else if s.ReportPeriodMs <= 10240 {
		log.Infof("received period %v; set RtPeriodIe_RT_PERIOD_IE_MS10240 period", s.ReportPeriodMs)
		return 19
	} else {
		log.Warnf("received period %v; too much long value - set to the maximum period RtPeriodIe_RT_PERIOD_IE_MS10240", s.ReportPeriodMs)
		return 19
	}
}

func (s *E2Session) createEventTriggerData() []byte {

	// Hardcoded just for test
	e2SmKpmEventTriggerDefinition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(s.getReportPeriod())
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

	err = client.Subscribe(ctx, subReq, ch)
	if err != nil {
		log.Error("Can't send SubscriptionRequest message")
		return err
	}

	log.Infof("Start forwarding Indication message to KPIMON controller")
	for indMsg := range ch {
		indChan <- indMsg
	}

	return nil
}
