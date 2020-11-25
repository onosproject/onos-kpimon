// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package admin

import (
	"context"
	"github.com/onosproject/onos-e2t/api/admin/v1"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
	"google.golang.org/grpc"
	"io"
	"time"
)

var log = logging.GetLogger("sb-admin")

// RicAPIAdminSesson is responsibnle for mapping connections to and interactions with the northbound admin API of ONOS-E2T
type RicAPIAdminSession struct {
	E2TEndpoint string
}

// NewSession creates a new admin southbound session of ONOS-KPIMON
func NewSession(e2tEndpoint string) *RicAPIAdminSession {
	log.Info("Creating RicAPIAdminSession")
	return &RicAPIAdminSession{
		E2TEndpoint: e2tEndpoint,
	}
}

// GetListE2NodeIDs returns the list of E2NodeIDs which are connected to ONOS-RIC
func (s *RicAPIAdminSession) GetListE2NodeIDs() ([]string, error) {
	var nodeIDs []string

	adminClient, err := s.oneTimeConnectionHandler()
	if err != nil {
		return []string{}, err
	}

	e2NodeIDStream, err := adminClient.ListE2NodeConnections(context.Background(), &admin.ListE2NodeConnectionsRequest{})
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

func (s *RicAPIAdminSession) oneTimeConnectionHandler() (admin.E2TAdminServiceClient, error) {
	log.Infof("Connecting to ONOS-E2T ... %S", s.E2TEndpoint)

	opts := []grpc.DialOption {
		grpc.WithStreamInterceptor(southbound.RetryingStreamClientInterceptor(100 * time.Microsecond)),
	}

	conn, err := southbound.Connect(context.Background(), s.E2TEndpoint, "", "", opts...)
	if err != nil {
		log.Errorf("Failed to connect: %s", err)
		return nil, err
	}

	log.Infof("Connected to %s", s.E2TEndpoint)

	adminClient := admin.NewE2TAdminServiceClient(conn)
	return adminClient, nil

}
