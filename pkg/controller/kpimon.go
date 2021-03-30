// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"sync"
)

var log = logging.GetLogger("controller", "kpimon")

// NewKpiMonController makes a new kpimon controller
func NewKpiMonController(indChan chan indication.Indication, smVersion string) KpiMonController {
	var kpimonController KpiMonController
	if smVersion == "v1" {
		kpimonController = newV1KpiMonController(indChan)
	} else if smVersion == "v2" {
		kpimonController = newV2KpiMonController(indChan)
	} else {
		// It shouldn't be hit
		log.Fatal("The received service model version %s is not valid - it must be v1 or v2", smVersion)
	}
	return kpimonController
}

// KpiMonController is an interface of the controller for KPIMON
type KpiMonController interface {
	Run()
	listenIndChan()
	parseIndMsg(indication.Indication)
}

// AbstractKpiMonController is an abstract struct for kpimon controller
type AbstractKpiMonController struct {
	KpiMonController
	IndChan       chan indication.Indication
	KpiMonResults map[KpiMonMetricKey]KpiMonMetricValue
	KpiMonMutex   sync.RWMutex
}

// CellIdentity is the ID for each cell
type CellIdentity struct {
	PlmnID string
	ECI    string
}

// KpiMonMetricKey is the key of monitoring result map
type KpiMonMetricKey struct {
	cellIdentity CellIdentity
	Metric       string
}

// KpiMonMetricValue is the value of monitoring result map
type KpiMonMetricValue struct {
	Value int32
}

func (c *AbstractKpiMonController) updateKpiMonResults(plmnID string, eci string, metricType string, metricValue int32) {
	key := KpiMonMetricKey{
		cellIdentity: CellIdentity{
			PlmnID: plmnID,
			ECI:    eci,
		},
		Metric: metricType,
	}
	value := KpiMonMetricValue{
		Value: metricValue,
	}
	c.KpiMonResults[key] = value

	log.Infof("KpiMonResults: %v", c.KpiMonResults)
}
