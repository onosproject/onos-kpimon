// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
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
	RicRequest     types.RicRequest
	RanFuncID      types.RanFunctionID
	ReportPeriodMs uint64
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string, e2subEndpoint string, ricActionID int32, ricRequestorID int32, ricInstanceID int32, ranFuncID uint8, reportPeriodMs uint64) *E2Session {
	log.Info("Creating RicAPIE2Session")
	return &E2Session{
		E2SubEndpoint: e2subEndpoint,
		E2TEndpoint:   e2tEndpoint,
		RicActionID:   types.RicActionID(ricActionID),
		RanFuncID:     types.RanFunctionID(ranFuncID),
		RicRequest: types.RicRequest{
			InstanceID:  types.RicInstanceID(ricInstanceID),
			RequestorID: types.RicRequestorID(ricRequestorID),
		},
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

func (s *E2Session) getReportPeriod() subscription.TimeToWait {
	var period subscription.TimeToWait

	switch s.ReportPeriodMs {
	case 0:
		period = subscription.TimeToWait_TIME_TO_WAIT_ZERO
	case 1:
		period = subscription.TimeToWait_TIME_TO_WAIT_W1MS
	case 2:
		period = subscription.TimeToWait_TIME_TO_WAIT_W2MS
	case 5:
		period = subscription.TimeToWait_TIME_TO_WAIT_W5MS
	case 10:
		period = subscription.TimeToWait_TIME_TO_WAIT_W10MS
	case 20:
		period = subscription.TimeToWait_TIME_TO_WAIT_W20MS
	case 30:
		period = subscription.TimeToWait_TIME_TO_WAIT_W30MS
	case 40:
		period = subscription.TimeToWait_TIME_TO_WAIT_W40MS
	case 50:
		period = subscription.TimeToWait_TIME_TO_WAIT_W50MS
	case 100:
		period = subscription.TimeToWait_TIME_TO_WAIT_W100MS
	case 200:
		period = subscription.TimeToWait_TIME_TO_WAIT_W200MS
	case 500:
		period = subscription.TimeToWait_TIME_TO_WAIT_W500MS
	case 1000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W1S
	case 2000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W2S
	case 5000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W5S
	case 10000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W10S
	case 20000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W20S
	case 60000:
		period = subscription.TimeToWait_TIME_TO_WAIT_W60S
	default:
		log.Warnf("period should be one of {0, 1, 2, 5, 10, 20, 30, 40, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 60000}ms,"+
			"%v is not valid; period is set to default period 0ms", s.ReportPeriodMs)
		period = subscription.TimeToWait_TIME_TO_WAIT_ZERO
	}

	return period
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
				Data:     []byte{},
			},
		},
		Actions: []subscription.Action{
			{
				ID:   int32(s.RicActionID),
				Type: subscription.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &subscription.SubsequentAction{
					Type:       subscription.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: s.getReportPeriod(),
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
