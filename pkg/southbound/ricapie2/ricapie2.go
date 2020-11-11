// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

import (
	"context"
	ricapie2 "github.com/onosproject/onos-e2t/api/ricapi/e2/v1beta1"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"google.golang.org/grpc"
	"io"
	"time"
)

var log = logging.GetLogger("sb-ricapie2")

// RicAPIE2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T
type RicAPIE2Session struct {
	E2TEndpoint string
	E2TClient   ricapie2.E2TServiceClient
}

// NewSession creates a new southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string) (*RicAPIE2Session, error) {
	log.Info("Creating RicAPIE2Session")
	return &RicAPIE2Session{
		E2TEndpoint: e2tEndpoint,
	}, nil
}

// Run starts the southbound control loop for handover.
func (s *RicAPIE2Session) Run() {
	log.Info("Started KPIMON Southbound session")
	go s.manageConnections()
	for {
		time.Sleep(100 * time.Second)
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

	s.streamHandler()
}

func (s *RicAPIE2Session) streamHandler() error {
	log.Info("Start ONOS-KPIMON App registration to ONOS-E2T for subscription\n")
	stream, err := s.E2TClient.Stream(context.Background())
	if err != nil {
		log.Errorf("Error on opening App stream connection %v", err)
		return err
	}

	s.subscribeE2T(stream)
	s.watchE2IndicationMsgs(stream)

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
			// TODO: indication message dispatcher/handler should be called here
		}
	}
}
