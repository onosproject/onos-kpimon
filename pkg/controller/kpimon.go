// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2sm_kpm_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
	"sync"
)

var log = logging.GetLogger("ctrl-kpimon")

// KpiMonCtrl is the controller for the KPI monitoring
type KpiMonCtrl struct {
	IndChan       chan indication.Indication
	KpiMonResults map[CellIdentity]int32
	KpiMonMutex	  sync.RWMutex
}

// CellIdentity is the ID for each cell
type CellIdentity struct {
	CuCpName string
	PlmnID   string
	NodeID   string
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
	var plmnID string
	var nodeID string
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

		log.Debugf("ind Header: %v", indHeader.GetIndicationHeaderFormat1())
		log.Debugf("E2SMKPM Ind Header: %v", indHeader.GetE2SmKpmIndicationHeader())

		if c.hasENbID(indHeader.GetIndicationHeaderFormat1()) {
			log.Debugf("eNB field: %v", 		indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().String())
			plmnID, nodeID, err = c.parseHeaderENbID(indHeader.GetIndicationHeaderFormat1())

			log.Debugf("PLMNID: %v", plmnID)
			log.Debugf("eNBID: %v", nodeID)
		} else if c.hasGNbID(indHeader.GetIndicationHeaderFormat1()) {
			log.Debugf("gNB field: %v", 		indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetGNb().String())
			plmnID, nodeID, err = c.parseHeaserGNbID(indHeader.GetIndicationHeaderFormat1())

			log.Debugf("PLMNID: %v", plmnID)
			log.Debugf("gNBID: %v", nodeID)
		} else {
			// TODO: have to support other types of ID for the future
			log.Errorf("The message header %v does not support yet. As of now, KPIMON supports both eNB and gNB ID type", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId())
		}

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

		c.KpiMonMutex.Lock()
		c.updateKpiMonResults(plmnID, nodeID,
			indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetGNbCuCpName().GetValue(),
			indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes())
		c.KpiMonMutex.Unlock()
	}
}

func (c *KpiMonCtrl) parseHeaderENbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, enbID string

	plmnID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().Value)
	enbID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId().Value)
	return plmnID, enbID, nil
}

func (c *KpiMonCtrl) parseHeaserGNbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, gnbID string

	plmnID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetGNb().GetGlobalGNbId().GetPlmnId().Value)
	gnbID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetGNb().GetGlobalGNbId().GetGnbId().GetGnbId().Value)

	return plmnID, gnbID, nil
}

func (c *KpiMonCtrl) hasENbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) bool {
	return header.GetIdGlobalKpmnodeId().GetENb() != nil
}

func (c *KpiMonCtrl) hasGNbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) bool {
	return header.GetIdGlobalKpmnodeId().GetGNb() != nil
}

func (c *KpiMonCtrl) updateKpiMonResults(plmnID string, nodeID string, cucpName string, numActiveUEs int32) {
	c.KpiMonResults[CellIdentity{CuCpName: cucpName, PlmnID: plmnID, NodeID: nodeID}] = numActiveUEs

	log.Infof("KpiMonResults: %v", c.KpiMonResults)
}
