// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
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
	Run(map[int]string)
	GetKpiMonResults() map[KpiMonMetricKey]KpiMonMetricValue
	GetKpiMonMutex() *sync.RWMutex
	SetGranularityPeriod(uint64)
	listenIndChan()
	parseIndMsg(indication.Indication)
	flushResultMap(string, string, string)
}

// AbstractKpiMonController is an abstract struct for kpimon controller
type AbstractKpiMonController struct {
	KpiMonController
	IndChan         chan indication.Indication
	KpiMonResults   map[KpiMonMetricKey]KpiMonMetricValue
	KpiMonMutex     sync.RWMutex
	KpiMonMetricMap map[int]string
	GranulPeriod    uint64
}

// CellIdentity is the ID for each cell
type CellIdentity struct {
	PlmnID string
	ECI    string
	CellID string
}

// KpiMonMetricKey is the key of monitoring result map
type KpiMonMetricKey struct {
	CellIdentity CellIdentity
	Timestamp    uint64
	Metric       string
}

// KpiMonMetricValue is the value of monitoring result map
type KpiMonMetricValue struct {
	Value string
}

func (c *AbstractKpiMonController) updateKpiMonResults(cellID string, plmnID string, eci string, metricType string, metricValue int32, timestamp uint64) {
	key := KpiMonMetricKey{
		CellIdentity: CellIdentity{
			PlmnID: plmnID,
			ECI:    eci,
			CellID: cellID,
		},
		Metric:    metricType,
		Timestamp: timestamp,
	}
	value := KpiMonMetricValue{
		Value: fmt.Sprintf("%d", metricValue),
	}
	c.KpiMonResults[key] = value

	log.Debugf("KpiMonResults: %v", c.KpiMonResults)
}

// GetKpiMonMutex returns Mutex to lock and unlock kpimon result map
func (c *AbstractKpiMonController) GetKpiMonMutex() *sync.RWMutex {
	return &c.KpiMonMutex
}

// GetKpiMonResults returns kpimon result map for all keys
func (c *AbstractKpiMonController) GetKpiMonResults() map[KpiMonMetricKey]KpiMonMetricValue {
	return c.KpiMonResults
}

// SetGranularityPeriod returns the granularity period
func (c *AbstractKpiMonController) SetGranularityPeriod(granularity uint64) {
	c.GranulPeriod = granularity
}

// flushResultMap flushes Reuslt map - carefully use it: have to lock before we call this
func (c *AbstractKpiMonController) flushResultMap(cellID string, plmnID string, eci string) {
	for k := range c.KpiMonResults {
		if k.CellIdentity.CellID == cellID && k.CellIdentity.PlmnID == plmnID && k.CellIdentity.ECI == eci {
			delete(c.KpiMonResults, k)
		}
	}
}
