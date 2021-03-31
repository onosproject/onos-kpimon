// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
)

func newV2KpiMonController(indChan chan indication.Indication) *V2KpiMonController {
	return &V2KpiMonController{
		AbstractKpiMonController: &AbstractKpiMonController{
			IndChan:       indChan,
			KpiMonResults: make(map[KpiMonMetricKey]KpiMonMetricValue),
		},
	}
}

// V2KpiMonController is the kpimon controller for KPM v2.0
type V2KpiMonController struct {
	*AbstractKpiMonController
}

// Run runs the kpimon controller for KPM v2.0
func (v2 *V2KpiMonController) Run() {
	v2.listenIndChan()
}

func (v2 *V2KpiMonController) listenIndChan() {
	for indMsg := range v2.IndChan {
		v2.parseIndMsg(indMsg)
	}
}

func (v2 *V2KpiMonController) parseIndMsg(indMsg indication.Indication) {
	var plmnID string
	var eci string

	log.Debugf("Raw indication message: %v", indMsg)
	indHeader := e2sm_kpm_v2.E2SmKpmIndicationHeader{}
	err := proto.Unmarshal(indMsg.Payload.Header, &indHeader)
	if err != nil {
		log.Errorf("Error - Unmarshalling header protobytes to struct: %v", err)
		return
	}

	log.Debugf("ind Header: %v", indHeader.GetIndicationHeaderFormat1())
	log.Debugf("E2SMKPM Ind Header: %v", indHeader.GetE2SmKpmIndicationHeader())

	plmnID, eci, _ = v2.getCellIdentitiesFromHeader(indHeader.GetIndicationHeaderFormat1())

	indMessage := e2sm_kpm_v2.E2SmKpmIndicationMessage{}
	err = proto.Unmarshal(indMsg.Payload.Message, &indMessage)
	if err != nil {
		log.Errorf("Error - Unmarshalling message protobytes to struct: %s", err)
		return
	}

	log.Debugf("ind Msgs: %v", indMessage.GetIndicationMessageFormat1())
	log.Debugf("E2SMKPM ind Msgs: %v", indMessage.GetE2SmKpmIndicationMessage())

	v2.KpiMonMutex.Lock()
	for i := 0; i < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[0].GetMeasRecord().GetValue()); i++ {
		metricValue := int32(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[0].GetMeasRecord().GetValue()[i].GetInteger())

		log.Debugf("Value in Indication message for type %v: %v", indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[i].GetMeasType().GetMeasName().GetValue(), metricValue)

		v2.updateKpiMonResults(plmnID, eci, indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[i].GetMeasType().GetMeasName().GetValue(), metricValue)
	}
	log.Debugf("KpiMonResult: %v", v2.KpiMonResults)
	v2.KpiMonMutex.Unlock()
}

func (v2 *V2KpiMonController) getCellIdentitiesFromHeader(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, eci string

	if (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetPLmnIdentity() != nil {
		plmnID = fmt.Sprintf("%d", utils.DecodePlmnIDToUint32((*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().GetValue()))
	} else if (*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetPlmnId() != nil {
		plmnID = fmt.Sprintf("%d", utils.DecodePlmnIDToUint32((*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetPlmnId().GetValue()))
	} else if (*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetPLmnIdentity() != nil {
		plmnID = fmt.Sprintf("%d", utils.DecodePlmnIDToUint32((*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetPLmnIdentity().GetValue()))
	} else if (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetPlmnId() != nil {
		plmnID = fmt.Sprintf("%d", utils.DecodePlmnIDToUint32((*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetPlmnId().GetValue()))
	} else {
		log.Errorf("Error when Parsing PLMN ID in indication message header - %v", header.GetKpmNodeId())
	}

	if (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId().GetValue())
	} else if (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetENbId().GetHomeENbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetENbId().GetHomeENbId().GetValue())
	} else if (*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetGnbId().GetGnbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetGnbId().GetGnbId().GetValue())
	} else if (*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetGNbId().GetGNbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetGNbId().GetGNbId().GetValue())
	} else if (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetLongMacroENbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetLongMacroENbId().GetValue())
	} else if (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetShortMacroENbId() != nil {
		eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetShortMacroENbId().GetValue())
	} else {
		log.Errorf("Error when Parsing ECI in indication message header - %v", header.GetKpmNodeId())
	}
	return plmnID, eci, nil
}
