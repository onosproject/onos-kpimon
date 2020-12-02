// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2apies"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/pdubuilder"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/encoding"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/node"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"
	"strconv"
	"strings"
)

var log = logging.GetLogger("sb-ricapie2")

// E2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type E2Session struct {
	E2SubEndpoint string
	E2TEndpoint   string
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string, e2subEndpoint string) *E2Session {
	log.Info("Creating RicAPIE2Session")
	return &E2Session{
		E2SubEndpoint: e2subEndpoint,
		E2TEndpoint:   e2tEndpoint,
	}
}

// Run starts the southbound to watch indication messages
func (s *E2Session) Run(indChan chan indication.Indication, adminSession *admin.E2AdminSession) {
	log.Info("Started KPIMON Southbound session")
	s.manageConnections(indChan, adminSession)
}

// manageConnections handles connections between ONOS-KPIMON and ONOS-E2T/E2Sub.
func (s *E2Session) manageConnections(indChan chan indication.Indication, adminSession *admin.E2AdminSession) {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)
	for {
		nodeIDs, err := adminSession.GetListE2NodeIDs()
		if err != nil {
			log.Errorf("Cannot get NodeIDs through Admin API: %s", err)
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

func (s *E2Session) createSubscriptionRequest(nodeID string) (subscription.Subscription, error) {
	ricActionsToBeSetup := make(map[types.RicActionID]types.RicActionDef)
	ricActionsToBeSetup[100] = types.RicActionDef{
		RicActionID:         100,
		RicActionType:       e2apies.RicactionType_RICACTION_TYPE_REPORT,
		RicSubsequentAction: e2apies.RicsubsequentActionType_RICSUBSEQUENT_ACTION_TYPE_CONTINUE,
		Ricttw:              e2apies.RictimeToWait_RICTIME_TO_WAIT_ZERO,
		RicActionDefinition: []byte{0x11, 0x22},
	}

	E2apPdu, err := pdubuilder.CreateRicSubscriptionRequestE2apPdu(types.RicRequest{RequestorID: 0, InstanceID: 0},
		0, nil, ricActionsToBeSetup)

	if err != nil {
		return subscription.Subscription{}, err
	}

	subReq := subscription.Subscription{
		EncodingType: encoding.PROTO,
		NodeID:       node.ID(nodeID),
		Payload: subscription.Payload{
			Value: E2apPdu,
		},
	}

	return subReq, nil
}

func (s *E2Session) subscribeE2T(indChan chan indication.Indication, nodeIDs []string) error {
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

	// Start to send Indication messages to the indChan which KPIMON Controller will subscribe
	for indMsg := range ch {
		indChan <- indMsg
	}

	return nil
}
