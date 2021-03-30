// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	"github.com/onosproject/onos-kpimon/pkg/controller"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

func newV1Manager(config Config) *V1Manager {
	log.Info("Creating Manager for KPM V1.0")
	indCh := make(chan indication.Indication)
	return &V1Manager{
		AbstractManager: &AbstractManager{
			Config: config,
			Chans: Channels{
				IndCh: indCh,
			},
			Sessions: SBSessions{
				AdminSession: admin.NewE2AdminSession(config.E2tEndpoint),
				E2Session:    ricapie2.NewE2Session(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID, 0, config.SMName, config.SMVersion),
			},
			Ctrls: Controllers{
				KpiMonController: controller.NewKpiMonController(indCh, config.SMVersion),
			},
		},
	}
}

// V1Manager is a KPIMON manager for KPM v1.0
type V1Manager struct {
	*AbstractManager
}
