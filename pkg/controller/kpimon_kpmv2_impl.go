// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"encoding/binary"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"google.golang.org/protobuf/proto"
	"strconv"
	"sync"
	"time"
)

func newV2KpiMonController(indChan chan indication.Indication, kpiMonMetricMap map[int]string,
	kpiMonMetricMapMutex *sync.RWMutex, cellIDMapForSub map[int64]*e2sm_kpm_v2.CellGlobalId, cellIDMapMutex *sync.RWMutex) KpiMonController {
	return &V2KpiMonController{
		AbstractKpiMonController: &AbstractKpiMonController{
			IndChan:              indChan,
			KpiMonResults:        make(map[KpiMonMetricKey]KpiMonMetricValue),
			KpiMonMetricMap:      kpiMonMetricMap,
			KpiMonMetricMapMutex: kpiMonMetricMapMutex,
			CellIDMapForSub:      cellIDMapForSub,
			CellIDMapMutex:       cellIDMapMutex,
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

	log.Debugf("Raw indication message: %v", indMsg)
	indHeader := e2sm_kpm_v2.E2SmKpmIndicationHeader{}
	err := proto.Unmarshal(indMsg.Payload.Header, &indHeader)
	if err != nil {
		log.Errorf("Error - Unmarshalling header protobytes to struct: %v", err)
		return
	}

	log.Debugf("ind Header: %v", indHeader.GetIndicationHeaderFormat1())
	log.Debugf("E2SMKPM Ind Header: %v", indHeader.GetE2SmKpmIndicationHeader())

	startTime := v2.getTimeStampFromHeader(indHeader.GetIndicationHeaderFormat1())

	startTimeUnix := time.Unix(int64(startTime), 0)
	startTimeUnixNano := startTimeUnix.UnixNano()

	log.Debugf("start timestamp: %d, %s (ns: %d / s: )", startTime, startTimeUnix, startTimeUnix.UnixNano(), startTimeUnix.Unix())

	indMessage := e2sm_kpm_v2.E2SmKpmIndicationMessage{}
	err = proto.Unmarshal(indMsg.Payload.Message, &indMessage)
	if err != nil {
		log.Errorf("Error - Unmarshalling message protobytes to struct: %s", err)
		return
	}

	log.Debugf("ind Msgs: %v", indMessage.GetIndicationMessageFormat1())
	log.Debugf("E2SMKPM ind Msgs: %v", indMessage.GetE2SmKpmIndicationMessage())

	subID := indMessage.GetIndicationMessageFormat1().GetSubscriptId().GetValue()
	v2.CellIDMapMutex.RLock()
	plmnID, err := v2.getPlmnID(v2.CellIDMapForSub[subID])
	if err != nil {
		log.Errorf("%v", err)
	}
	eci, err := v2.getEci(v2.CellIDMapForSub[subID])
	if err != nil {
		log.Errorf("%v", err)
	}
	v2.CellIDMapMutex.RUnlock()
	log.Debugf("PLMNID (%v) and ECI (%v) for subscription ID (%v)", plmnID, eci, subID)

	v2.KpiMonMutex.Lock()
	var cid string
	if indMessage.GetIndicationMessageFormat1().GetCellObjId() == nil {
		plmnIDInt, err := strconv.Atoi(plmnID)
		if err != nil {
			log.Errorf("Failed to convert plmnid (%s) from string to int: %v", plmnID, err)
		}

		eciInt, err := strconv.Atoi(eci)
		if err != nil {
			log.Errorf("Failed to convert eci (%s) from string to int: %v", eci, err)
		}
		cid = fmt.Sprintf("%d", types.ToECGI(types.PlmnID(plmnIDInt), types.ECI(eciInt)))
	} else {
		cid = indMessage.GetIndicationMessageFormat1().GetCellObjId().Value
	}

	v2.flushResultMap(cid, plmnID, eci)
	for i := 0; i < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()); i++ {
		for j := 0; j < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()); j++ {
			metricValue := int32(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()[j].GetInteger())
			tmpTimestamp := uint64(startTimeUnixNano) + v2.GranulPeriod*uint64(1000000)*uint64(i)
			log.Debugf("Timestamp for %d-th element: %v", i, tmpTimestamp)
			if indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue() == "" {
				v2.KpiMonMetricMapMutex.RLock()
				log.Debugf("Indication message does not have MeasName - use MeasID")
				log.Debugf("Value in Indication message for type %v (MeasID-%d): %v", v2.KpiMonMetricMap[int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value)], int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value), metricValue)
				v2.updateKpiMonResults(cid, plmnID, eci, v2.KpiMonMetricMap[int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value)], metricValue, tmpTimestamp)
				v2.KpiMonMetricMapMutex.RUnlock()
			} else {
				log.Debugf("Value in Indication message for type %v: %v", indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue(), metricValue)
				v2.updateKpiMonResults(cid, plmnID, eci, indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue(), metricValue, tmpTimestamp)
			}
		}
	}
	//log.Debugf("KpiMonResult: %v", v2.KpiMonResults)
	v2.KpiMonMutex.Unlock()
}

func (v2 *V2KpiMonController) getTimeStampFromHeader(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) uint64 {
	timeBytes := (*header).GetColletStartTime().Value
	timeInt32 := binary.BigEndian.Uint32(timeBytes)
	return uint64(timeInt32)
}

func (v2 *V2KpiMonController) getPlmnID(cellGlobalID *e2sm_kpm_v2.CellGlobalId) (string, error) {
	if cellGlobalID.GetNrCgi().GetPLmnIdentity() != nil {
		return fmt.Sprintf("%d", utils.DecodePlmnIDToUint32(cellGlobalID.GetNrCgi().GetPLmnIdentity().GetValue())), nil
	} else if cellGlobalID.GetEUtraCgi().GetPLmnIdentity() != nil {
		return fmt.Sprintf("%d", utils.DecodePlmnIDToUint32(cellGlobalID.GetEUtraCgi().GetPLmnIdentity().GetValue())), nil
	} else {
		return "", fmt.Errorf("PLMN ID cannot be decoded - cell global ID: %v", cellGlobalID)
	}
}

func (v2 *V2KpiMonController) getEci(cellGlobalID *e2sm_kpm_v2.CellGlobalId) (string, error) {
	if cellGlobalID.GetNrCgi().GetPLmnIdentity() != nil {
		return fmt.Sprintf("%d", cellGlobalID.GetNrCgi().GetNRcellIdentity().GetValue().GetValue()), nil
	} else if cellGlobalID.GetEUtraCgi().GetPLmnIdentity() != nil {
		return fmt.Sprintf("%d", cellGlobalID.GetEUtraCgi().GetEUtracellIdentity().GetValue().GetValue()), nil
	} else {
		return "", fmt.Errorf("ECI cannot be decoded - cell global ID: %v", cellGlobalID)
	}
}
