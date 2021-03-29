// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

const (
	RRC_ConnEstabAtt_Tot = "RRC.ConnEstabAtt.Tot"
	RRC_ConnEstabSucc_Tot = "RRC.ConnEstabSucc.Tot"
	RRC_ConnReEstabAtt_Tot = "RRC.ConnReEstabAtt.Tot"
	RRC_ConnReEstabAtt_reconfigFail = "RRC.ConnReEstabAtt.reconfigFail"
	RRC_ConnReEstabAtt_HOFail = "RRC.ConnReEstabAtt.HOFail"
	RRC_ConnReEstabAtt_Other = "RRC.ConnReEstabAtt.Other"
	RRC_Conn_Avg = "RRC.Conn.Avg"
	RRC_Conn_Max = "RRC.Conn.Max"
)

func newV2KpiMonController(indChan chan indication.Indication) *V2KpiMonController {
	return &V2KpiMonController{
		AbstractKpiMonController: &AbstractKpiMonController{
			IndChan: indChan,
			KpiMonResults: make(map[KpiMonMetricKey]KpiMonMetricValue),
		},
	}
}

type V2KpiMonController struct {
	*AbstractKpiMonController
}

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
	for i := 0; i < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()); i++ {
		var metricValue int32
		if len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()) == 0 {
			metricValue = 0
		} else if indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()[0].GetNoValue() == 0 {
			metricValue = 0
		}

		metricValue = int32(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()[0].GetInteger())
		switch indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[i].GetMeasType().GetMeasName().GetValue() {
		case RRC_ConnEstabAtt_Tot:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnEstabAtt_Tot, metricValue)
			break
		case RRC_ConnEstabSucc_Tot:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnEstabSucc_Tot, metricValue)
			break
		case RRC_ConnReEstabAtt_Tot:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnReEstabAtt_Tot, metricValue)
			break
		case RRC_ConnReEstabAtt_reconfigFail:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnReEstabAtt_reconfigFail, metricValue)
			break
		case RRC_ConnReEstabAtt_HOFail:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnReEstabAtt_HOFail, metricValue)
			break
		case RRC_ConnReEstabAtt_Other:
			v2.updateKpiMonResults(plmnID, eci, RRC_ConnReEstabAtt_Other, metricValue)
			break
		case RRC_Conn_Avg:
			v2.updateKpiMonResults(plmnID, eci, RRC_Conn_Avg, metricValue)
			break
		case RRC_Conn_Max:
			v2.updateKpiMonResults(plmnID, eci, RRC_Conn_Max, metricValue)
			break
		default:
			log.Warnf("Unknown MeasName: %v", indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[i].GetMeasType().GetMeasName().GetValue())
			break
		}
	}
	log.Debugf("KpiMonResult: %v", v2.KpiMonResults)
	v2.KpiMonMutex.Unlock()
}

func (v2 *V2KpiMonController) getCellIdentitiesFromHeader(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, eci string

	if (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetPLmnIdentity() != nil {
		plmnID = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().GetValue())
	} else if (*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetPlmnId() != nil {
		plmnID = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetGNb().GetGlobalGNbId().GetPlmnId().GetValue())
	} else if (*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetPLmnIdentity() != nil {
		plmnID = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetEnGNb().GetGlobalGNbId().GetPLmnIdentity().GetValue())
	} else if (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetPlmnId() != nil {
		plmnID = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetNgENb().GetGlobalNgENbId().GetPlmnId().GetValue())
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

func (v2 *V2KpiMonController) parseHeaderMacroEnbID(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
	var plmnID, eci string
	plmnID = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetPLmnIdentity().GetValue())
	eci = fmt.Sprintf("%d", (*header).GetKpmNodeId().GetENb().GetGlobalENbId().GetENbId().GetMacroENbId().GetValue())
	return plmnID, eci, nil
}
