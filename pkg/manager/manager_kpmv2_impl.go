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

func newV2Manager(config Config) *V2Manager {
	log.Info("Creating Manager for KPM V2.0")
	indCh := make(chan indication.Indication)
	kpiMonMetricMap := make(map[int]string)
	return &V2Manager{
		AbstractManager: &AbstractManager{
			Config: config,
			Chans: Channels{
				IndCh: indCh,
			},
			Sessions: SBSessions{
				AdminSession: admin.NewE2AdminSession(config.E2tEndpoint),
				E2Session:    ricapie2.NewE2Session(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID, 0, config.SMName, config.SMVersion, kpiMonMetricMap),
			},
			Ctrls: Controllers{
				KpiMonController: controller.NewKpiMonController(indCh, config.SMVersion),
			},
			Maps: Maps{
				KpiMonMetricMap: kpiMonMetricMap,
			},
		},
	}
}

// V2Manager is a KPIMON manager for KPM v2.0
type V2Manager struct {
	*AbstractManager
}
