// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2sm_kpm_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
	"time"
)

func newV1KpiMonController(indChan chan indication.Indication) *V1KpiMonController {
	return &V1KpiMonController{
		AbstractKpiMonController: &AbstractKpiMonController{
			IndChan:       indChan,
			KpiMonResults: make(map[KpiMonMetricKey]KpiMonMetricValue),
		},
	}
}

// V1KpiMonController is the kpimon controller for KPM v1.0
type V1KpiMonController struct {
	*AbstractKpiMonController
}

// Run runs the kpimon controller for KPM v1.0
func (v1 *V1KpiMonController) Run(kpimonMetricMap map[int]string) {
	v1.KpiMonMetricMap = kpimonMetricMap
	v1.listenIndChan()
}

func (v1 *V1KpiMonController) listenIndChan() {
	for indMsg := range v1.IndChan {
		v1.parseIndMsg(indMsg)
	}
}

func (v1 *V1KpiMonController) parseIndMsg(indMsg indication.Indication) {
	var plmnID string
	var eci string
	log.Debugf("Raw indication message: %v", indMsg)

	indHeader := e2sm_kpm_ies.E2SmKpmIndicationHeader{}
	err := proto.Unmarshal(indMsg.Payload.Header, &indHeader)
	if err != nil {
		log.Errorf("Error - Unmarshalling header protobytes to struct: %v", err)
		return
	}

	log.Debugf("ind Header: %v", indHeader.GetIndicationHeaderFormat1())
	log.Debugf("E2SMKPM Ind Header: %v", indHeader.GetE2SmKpmIndicationHeader())

	if v1.hasENbID(indHeader.GetIndicationHeaderFormat1()) {
		log.Debugf("eNB field: %v", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetENb().String())
		plmnID, eci, _ = v1.parseHeaderENbID(indHeader.GetIndicationHeaderFormat1())

		log.Debugf("PLMNID: %v", plmnID)
		log.Debugf("eNBID: %v", eci)
	} else if v1.hasGNbID(indHeader.GetIndicationHeaderFormat1()) {
		log.Debugf("gNB field: %v", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId().GetGNb().String())
		plmnID, eci, _ = v1.parseHeaserGNbID(indHeader.GetIndicationHeaderFormat1())

		log.Debugf("PLMNID: %v", plmnID)
		log.Debugf("gNBID: %v", eci)
	} else {
		// TODO: have to support other types of ID for the future
		log.Errorf("The message header %v does not support yet. As of now, KPIMON supports both eNB and gNB ID type", indHeader.GetIndicationHeaderFormat1().GetIdGlobalKpmnodeId())
	}

	indMessage := e2sm_kpm_ies.E2SmKpmIndicationMessage{}
	err = proto.Unmarshal(indMsg.Payload.Message, &indMessage)
	if err != nil {
		log.Errorf("Error - Unmarshalling message protobytes to struct: %s", err)
		return
	}

	log.Debugf("ind Msgs: %v", indMessage.GetIndicationMessageFormat1())
	log.Debugf("E2SMKPM ind Msgs: %v", indMessage.GetE2SmKpmIndicationMessage())

	// allow pmContainers array being empty
	if len(indMessage.GetIndicationMessageFormat1().GetPmContainers()) == 0 {
		log.Warnf("PmContainers array field in indication message is empty")
		return
	}
	log.Debugf("numUEs: %v", indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes())
	v1.KpiMonMutex.Lock()
	v1.flushResultMap("N/A", plmnID, eci)
	v1.updateKpiMonResults("N/A", plmnID, eci, "numActiveUEs",
		indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes(), uint64(time.Now().UnixNano()))
	v1.KpiMonMutex.Unlock()
}

func (v1 *V1KpiMonController) parseHeaderENbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, enbID string

	plmnID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().Value)
	enbID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId().Value)
	return plmnID, enbID, nil
}

func (v1 *V1KpiMonController) parseHeaserGNbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, gnbID string

	plmnID = fmt.Sprintf("%d", utils.DecodePlmnIDToUint32((*header).GetIdGlobalKpmnodeId().GetGNb().GetGlobalGNbId().GetPlmnId().Value))
	gnbID = fmt.Sprintf("%d", (*header).GetIdGlobalKpmnodeId().GetGNb().GetGlobalGNbId().GetGnbId().GetGnbId().Value)

	return plmnID, gnbID, nil
}

func (v1 *V1KpiMonController) hasENbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) bool {
	return header.GetIdGlobalKpmnodeId().GetENb() != nil
}

func (v1 *V1KpiMonController) hasGNbID(header *e2sm_kpm_ies.E2SmKpmIndicationHeaderFormat1) bool {
	return header.GetIdGlobalKpmnodeId().GetGNb() != nil
}
