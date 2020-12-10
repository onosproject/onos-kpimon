// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package main

import (
	"flag"

	"github.com/onosproject/onos-kpimon/pkg/manager"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("main")

func main() {
	caPath := flag.String("caPath", "", "path to CA certificate")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	e2tEndpoint := flag.String("e2tEndpoint", "onos-e2t:5150", "E2T service endpoint")
	e2subEndpoint := flag.String("e2subEndpoint", "onos-e2sub:5150", "E2Sub service endpoint")
	ricActionID := flag.Int("ricActionID", 10, "RIC Action ID in E2 message")
	ricRequestorID := flag.Int("ricRequestorID", 10, "RIC Requestor ID in E2 message")
	ricInstanceID := flag.Int("ricInstanceID", 1, "RIC Instance ID in E2 message")
	ranFuncID := flag.Uint("ranFuncID", 1, "RAN Function ID in E2 message")

	ready := make(chan bool)

	flag.Parse()

	_, err := certs.HandleCertPaths(*caPath, *keyPath, *certPath, true)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting onos-kpimon")
	cfg := manager.Config{
		CAPath:         *caPath,
		KeyPath:        *keyPath,
		CertPath:       *certPath,
		E2tEndpoint:    *e2tEndpoint,
		E2SubEndpoint:  *e2subEndpoint,
		GRPCPort:       5150,
		RicActionID:    int32(*ricActionID),
		RicRequestorID: int32(*ricRequestorID),
		RicInstanceID:  int32(*ricInstanceID),
		RanFuncID:      uint8(*ranFuncID),
	}

	mgr := manager.NewManager(cfg)
	mgr.Run()
	<-ready
}
