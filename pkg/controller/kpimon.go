// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	e2sm_kpm_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("ctrl-kpimon")

// KpiMonCtrl is the controller for the KPI monitoring
type KpiMonCtrl struct {
	IndChan       chan indication.Indication
	KpiMonResults map[CellIdentity]int32
}

// CellIdentity is the ID for each cell
type CellIdentity struct {
	CuCpName string
	PlmnID   string
	CellID   string
}

// NewKpiMonController creates a new KpiMonController
func NewKpiMonController(indChan chan indication.Indication) *KpiMonCtrl {
	log.Info("Start ONOS-KPIMON Application Controller")
	return &KpiMonCtrl{
		IndChan:       indChan,
		KpiMonResults: make(map[CellIdentity]int32),
	}
}

// Run function runs to KpiMonController
func (c *KpiMonCtrl) Run() {
	c.listenIndChan()
}

// listenIndChan is the function to listen indication message channel
func (c *KpiMonCtrl) listenIndChan() {
	var err error
	for indMsg := range c.IndChan {
		indHeaderByte := indMsg.Payload.Header
		indMessageByte := indMsg.Payload.Message

		log.Debugf("Raw msg: %v", indMsg)

		indHeader := e2sm_kpm_ies.E2SmKpmIndicationHeader{}
		err = proto.Unmarshal(indHeaderByte, &indHeader)
		if err != nil {
			log.Errorf("Error - Unmarshalling header protobytes to struct: %s", err)
			continue
		}

		// TODO: have to add handler for the situation when gNB comes in - nil pointer exception should happen.
		log.Debugf("ind Header: %v", indHeader.GetIndicationHeaderFormat1())
		log.Debugf("E2SMKPM Ind Header: %v", indHeader.GetE2SmKpmIndicationHeader())
		log.Debugf("PLMNID: %v", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().Value)
		log.Debugf("eNBID: %v", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId().Value)

		indMessage := e2sm_kpm_ies.E2SmKpmIndicationMessage{}
		err = proto.Unmarshal(indMessageByte, &indMessage)
		if err != nil {
			log.Errorf("Error - Unmarshalling message protobytes to struct: %s", err)
			continue
		}

		log.Debugf("ind Msgs: %v", indMessage.GetIndicationMessageFormat1())
		log.Debugf("E2SMKPM ind Msgs: %v", indMessage.GetE2SmKpmIndicationMessage())

		// allow pmContainers array being empty
		if len(indMessage.GetIndicationMessageFormat1().GetPmContainers()) == 0 {
			log.Warnf("PmContainers array field in indication message is empty")
			continue
		}

		log.Debugf("numUEs: %v", indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes())
		log.Debugf("CUCP Name: %v", indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetGNbCuCpName().GetValue())

		c.updateKpiMonResults(indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetPLmnIdentity(),
			indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId(),
			indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetGNbCuCpName().GetValue(),
			indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes())
	}
}

func (c *KpiMonCtrl) updateKpiMonResults(plmnID *e2sm_kpm_ies.PlmnIdentity, cellID *e2sm_kpm_ies.BitString, cucpName string, numActiveUEs int32) {
	strPlmnID := fmt.Sprintf("%d", (*plmnID).Value)
	strCellID := fmt.Sprintf("%d", (*cellID).Value)
	c.KpiMonResults[CellIdentity{CuCpName: cucpName, PlmnID: strPlmnID, CellID: strCellID}] = numActiveUEs

	log.Infof("KpiMonResults: %v", c.KpiMonResults)
}
