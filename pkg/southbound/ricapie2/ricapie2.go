// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("sb-ricapie2")

// E2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type E2Session struct {
	E2SubEndpoint	string
	E2TEndpoint		string
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string, e2subEndpoint string) *E2Session {
	log.Info("Creating RicAPIE2Session")
	return &E2Session{
		E2SubEndpoint: e2subEndpoint,
		E2TEndpoint: e2tEndpoint,
	}
}

// Run starts the southbound to watch indication messages
func (s *E2Session) Run(indChan chan indication.Indication, ricAPIAdminSession *admin.RicAPIAdminSession) {
	log.Info("Started KPIMON Southbound session")
}

// manageConnections handles connections between ONOS-KPIMON and ONOS-E2T/E2Sub.
func (s *E2Session) manageConnections() {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)

}

/*
// RicAPIE2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type RicAPIE2Session struct {
	E2TEndpoint string
	E2TClient   ricapie2.E2TServiceClient
	SubMsgTimer	int
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string) (*RicAPIE2Session, error) {
	log.Info("Creating RicAPIE2Session")
	return &RicAPIE2Session{
		E2TEndpoint: e2tEndpoint,
	}, nil
}

// Run starts the southbound control loop
func (s *RicAPIE2Session) Run() {
	log.Info("Started KPIMON Southbound session")
	go s.manageConnections()
	for {
		time.Sleep(100 * time.Second)
	}
}

func (s *RicAPIE2Session) createSubscriptionRequest(nodeID string) (subscription.Subscription, error) {
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

func (s *RicAPIE2Session) subscribeE2T(nodeID string, appID string) (chan indication.Indication, error) {

	e2SubHost := strings.Split(s.E2TEndpoint, ":")[0]
	e2SubPort, err := strconv.Atoi(strings.Split(s.E2TEndpoint, ":")[1])

	if err != nil {
		log.Error("onos-e2sub's port information or endpoint information is wrong.")
		return nil, err
	}

	clientConfig := e2client.Config{
		AppID: app.ID(appID),
		SubscriptionService: e2client.ServiceConfig{
			Host: e2SubHost,
			Port: e2SubPort,
		},
	}

	client, err := e2client.NewClient(clientConfig)
	if err != nil {
		log.Error("Can't open E2Client.")
		return nil, err
	}
	ch := make(chan indication.Indication)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subReq, err := s.createSubscriptionRequest(nodeID)
	if err != nil {
		log.Error("Can't create SubsdcriptionRequest message")
		return nil, err
	}

	err := client.Subscribe(ctx, subReq, ch)
	if err != nil {
		log.Error("Can't send SubscriptionRequest message")
		return nil, err
	}

	select {
	case indicationMsg := <- ch:
		log.Debugf("%s message arrives", indicationMsg.EncodingType.String())
	case <- time.After(time.Duration(s.SubMsgTimer) * time.Second):
		log.Errorf("Timeout: Subscription response message does not arrives in %d seconds", s.SubMsgTimer)
	}

}


// manageConnections handles connections between ONOS-KPIMON and ONOS-E2T.
func (s *RicAPIE2Session) manageConnections() {
	log.Infof("Connecting to ONOS-E2T...%s", s.E2TEndpoint)

	for {
		// Attempt to create connection to the RIC
		opts := []grpc.DialOption{
			grpc.WithStreamInterceptor(southbound.RetryingStreamClientInterceptor(100 * time.Millisecond)),
		}
		conn, err := southbound.Connect(context.Background(), s.E2TEndpoint, "", "", opts...)
		if err != nil {
			log.Errorf("Failed to connect: %s", err)
			continue
		}
		log.Infof("Connected to %s", s.E2TEndpoint)
		// If successful, manage this connection and don't return until it is
		// no longer valid and all related resources have been properly cleaned-up.
		s.manageConnection(conn)
		time.Sleep(100 * time.Millisecond) // need to be in 10ms - 100ms
	}
}

// manageConnection is responsible for managing a single connection between HO App and ONOS RAN subsystem.
func (s *RicAPIE2Session) manageConnection(conn *grpc.ClientConn) {

	s.E2TClient = ricapie2.NewE2TServiceClient(conn)
	if s.E2TClient == nil {
		log.Error("Unable to get gRPC NewE2TServiceClient")
		return
	}

	defer conn.Close()

	_ = s.streamHandler()
}

func (s *RicAPIE2Session) streamHandler() error {
	log.Info("Start ONOS-KPIMON App registration to ONOS-E2T for subscription\n")
	stream, err := s.E2TClient.Stream(context.Background())
	if err != nil {
		log.Errorf("Error on opening App stream connection %v", err)
		return err
	}

	_ = s.subscribeE2T(stream)
	_ = s.watchE2IndicationMsgs(stream)

	return nil
}

func (s *RicAPIE2Session) subscribeE2T(stream ricapie2.E2TService_StreamClient) error {
	// make subscription request message
	reqMsg := ricapie2.StreamRequest{
		AppID:      "ONOS-KPIMON",
		InstanceID: "1",
	}

	log.Info("Sent ONOS-KPIMON App stream request message to ONOS-E2T for the subscription\n")
	err := stream.Send(&reqMsg)
	if err != nil {
		return err
	}

	log.Info("Waiting until stream response message arrives from ONOS-E2T")
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx.Err())
			return ctx.Err()
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			log.Error("End of file error", err, resp)
			return nil
		} else if err != nil {
			log.Error(err)
			continue
		} else {
			log.Infof("Received ONOS-KPIMON App registration response message from ONOS-E2T"+
				"(header - type:%s,smID:%s,status:%s; body - payload:%s\n", resp.Header.EncodingType.String(),
				resp.Header.ServiceModelInfo.ServiceModelId, resp.Header.ResponseStatus, resp.Payload)
		}
	}
}

func (s *RicAPIE2Session) watchE2IndicationMsgs(stream ricapie2.E2TService_StreamClient) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx.Err())
			return ctx.Err()
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			log.Error("End of file error", err, resp)
			return nil
		} else if err != nil {
			log.Error(err)
			continue
		} else {
			log.Info("TODO: indication message dispatcher/handler should be called here")
			// TODO: indication message dispatcher/handler should be called here
		}
	}
}
*/