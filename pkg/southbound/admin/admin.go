// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package admin

import (
	"context"
	adminapi "github.com/onosproject/onos-api/go/onos/e2t/admin"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"google.golang.org/grpc"
	"io"
	"time"
)

var log = logging.GetLogger("southbound", "admin")

// E2AdminSession is responsible for mapping connections to and interactions with the northbound admin API of ONOS-E2T
type E2AdminSession interface {
	GetListE2NodeIDs() ([]string, error)
	ConnectionHandler() (adminapi.E2TAdminServiceClient, error)
}

// E2AdminSessionData is a struct having data for E2 admin session
type E2AdminSessionData struct {
	E2AdminSession
	E2TEndpoint string
}

// NewE2AdminSession generates an E2 admin session
func NewE2AdminSession(e2tEndpoint string) E2AdminSession {
	var e2AdminSession E2AdminSession
	log.Info("Creating E2Admin session")
	e2AdminSession = &E2AdminSessionData{
		E2TEndpoint: e2tEndpoint,
	}
	return e2AdminSession
}

// GetListE2NodeIDs returns all E2 node IDs connected to ONOS-E2T
func (s *E2AdminSessionData) GetListE2NodeIDs() ([]string, error) {
	var nodeIDs []string

	adminClient, err := s.ConnectionHandler()
	if err != nil {
		return []string{}, err
	}

	e2NodeIDStream, err := adminClient.ListE2NodeConnections(context.Background(), &adminapi.ListE2NodeConnectionsRequest{})
	if err != nil {
		log.Errorf("Failed to call ListE2NodeConnections")
		return []string{}, err
	}

	for {
		e2NodeIDStream, err := e2NodeIDStream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Errorf("Failed to get e2NodeID")
			return []string{}, err
		} else if e2NodeIDStream != nil {
			nodeIDs = append(nodeIDs, e2NodeIDStream.Id)
		}
	}

	return nodeIDs, nil
}

// ConnectionHandler is a handler to manage E2 admin session
func (s *E2AdminSessionData) ConnectionHandler() (adminapi.E2TAdminServiceClient, error) {
	log.Infof("Connecting to ONOS-E2T ... %s", s.E2TEndpoint)

	opts := []grpc.DialOption{
		grpc.WithStreamInterceptor(southbound.RetryingStreamClientInterceptor(100 * time.Microsecond)),
	}

	conn, err := southbound.Connect(context.Background(), s.E2TEndpoint, "", "", opts...)
	if err != nil {
		log.Errorf("Failed to connect: %s", err)
		return nil, err
	}

	log.Infof("Connected to %s", s.E2TEndpoint)

	adminClient := adminapi.NewE2TAdminServiceClient(conn)
	return adminClient, nil
}
