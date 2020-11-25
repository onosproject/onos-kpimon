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

type RicAPIE2Session struct {
	E2SubEndpoint	string
	E2TEndpoint		string
}

// RicAPIE2Session is responsible for mapping connections to and interactions with the northbound of ONOS-E2T and E2Sub
func NewSession(e2tEndpoint string, e2subEndpoint string) *RicAPIE2Session {
	log.Info("Creating RicAPIE2Session")
	return &RicAPIE2Session{
		E2SubEndpoint: e2subEndpoint,
		E2TEndpoint: e2tEndpoint,
	}
}

// Run starts the southbound to watch indication messages
func (s *RicAPIE2Session) Run(indChan chan indication.Indication, ricAPIAdminSession *admin.RicAPIAdminSession) {
	log.Info("Started KPIMON Southbound session")
}

// manageConnections handles connections between ONOS-KPIMON and ONOS-E2T/E2Sub.
func (s *RicAPIE2Session) manageConnections() {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)

}
