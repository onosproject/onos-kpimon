// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("ctrl-kpimon")

// KpiMonCtrl is the controller for the KPI monitoring
type KpiMonCtrl struct {
	IndChan chan indication.Indication
}

// NewKpiMonController creates a new KpiMonController
func NewKpiMonController(indChan chan indication.Indication) *KpiMonCtrl {
	log.Info("Start ONOS-KPIMON Application Controller")
	return &KpiMonCtrl{
		IndChan: indChan,
	}
}

// PrintMessages is the function to print all indication messages - for debugging as of now
func (c *KpiMonCtrl) PrintMessages() {
	for indMsg := range c.IndChan {
		log.Infof("Received message %s", indMsg)
	}
}
