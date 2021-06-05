// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"github.com/onosproject/onos-kpimon/pkg/controller"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"sync"
)

func newV2Manager(config Config) *V2Manager {
	log.Info("Creating Manager for KPM V2.0")
	indCh := make(chan indication.Indication)
	kpiMonMetricMap := make(map[int]string)
	kpiMonMetricMapMutex := &sync.RWMutex{}
	cellIDMapForSub := make(map[int64]*e2sm_kpm_v2.CellGlobalId)
	cellIDMapMutex := &sync.RWMutex{}
	return &V2Manager{
		AbstractManager: &AbstractManager{
			Config: config,
			Chans: Channels{
				IndCh: indCh,
			},
			Sessions: SBSessions{
				AdminSession: admin.NewE2AdminSession(config.E2tEndpoint),
				E2Session: ricapie2.NewE2Session(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID,
					0, config.SMName, config.SMVersion, kpiMonMetricMap, kpiMonMetricMapMutex, cellIDMapForSub, cellIDMapMutex),
			},
			Ctrls: Controllers{
				KpiMonController: controller.NewKpiMonController(indCh, config.SMVersion, kpiMonMetricMap, kpiMonMetricMapMutex, cellIDMapForSub, cellIDMapMutex),
			},
			Maps: Maps{
				KpiMonMetricMap: kpiMonMetricMap,
				CellIDMapForSub: cellIDMapForSub,
			},
			Mutex: Mutex{
				CellIDMapMutex:       cellIDMapMutex,
				KpiMonMetricMapMutex: kpiMonMetricMapMutex,
			},
		},
	}
}

// V2Manager is a KPIMON manager for KPM v2.0
type V2Manager struct {
	*AbstractManager
}
