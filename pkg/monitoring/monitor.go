// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package monitoring

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/onosproject/onos-api/go/onos/ransim/types"
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	appConfig "github.com/onosproject/onos-kpimon/pkg/config"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"google.golang.org/protobuf/proto"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("controller", "kpimon")

// NewMonitor makes a new kpimon monitor
func NewMonitor(indChan chan indication.Indication, appConfig *appConfig.AppConfig) *Monitor {
	controller := &Monitor{
		IndChan:       indChan,
		KpiMonResults: make(map[KpiMonMetricKey]KpiMonMetricValue),
		appConfig:     appConfig,
	}

	return controller
}

// Monitor monitor data structure
type Monitor struct {
	IndChan         chan indication.Indication
	KpiMonResults   map[KpiMonMetricKey]KpiMonMetricValue
	KpiMonMutex     sync.RWMutex
	KpiMonMetricMap map[int]string
	appConfig       *appConfig.AppConfig
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

func (m *Monitor) updateKpiMonResults(cellID string, plmnID string, eci string, metricType string, metricValue int32, timestamp uint64) {
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
	m.KpiMonResults[key] = value

	log.Debugf("KpiMonResults: %v", m.KpiMonResults)
}

// GetKpiMonMutex returns Mutex to lock and unlock kpimon result map
func (m *Monitor) GetKpiMonMutex() *sync.RWMutex {
	return &m.KpiMonMutex
}

// GetKpiMonResults returns kpimon result map for all keys
func (m *Monitor) GetKpiMonResults() map[KpiMonMetricKey]KpiMonMetricValue {
	return m.KpiMonResults
}

// flushResultMap flushes Reuslt map - carefully use it: have to lock before we call this
func (m *Monitor) flushResultMap(cellID string, plmnID string, eci string) {
	for k := range m.KpiMonResults {
		if k.CellIdentity.CellID == cellID && k.CellIdentity.PlmnID == plmnID && k.CellIdentity.ECI == eci {
			delete(m.KpiMonResults, k)
		}
	}
}

// Run runs the kpimon controller for KPM v2.0
func (m *Monitor) Run(kpimonMetricMap map[int]string) {
	m.KpiMonMetricMap = kpimonMetricMap
	m.listenIndChan()
}

func (m *Monitor) listenIndChan() {
	for indMsg := range m.IndChan {
		m.parseIndMsg(indMsg)
	}
}

func (m *Monitor) parseIndMsg(indMsg indication.Indication) {
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

	plmnID, eci, _ = m.getCellIdentitiesFromHeader(indHeader.GetIndicationHeaderFormat1())
	startTime := m.getTimeStampFromHeader(indHeader.GetIndicationHeaderFormat1())

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

	m.KpiMonMutex.Lock()
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

	m.flushResultMap(cid, plmnID, eci)
	for i := 0; i < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()); i++ {
		for j := 0; j < len(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()); j++ {
			metricValue := int32(indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()[i].GetMeasRecord().GetValue()[j].GetInteger())
			granularityPeriod, _ := m.appConfig.GetGranularityPeriod()
			tmpTimestamp := uint64(startTimeUnixNano) + granularityPeriod*uint64(1000000)*uint64(i)
			log.Debugf("Timestamp for %d-th element: %v", i, tmpTimestamp)
			if indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue() == "" {
				log.Debugf("Indication message does not have MeasName - use MeasID")
				log.Debugf("Value in Indication message for type %v (MeasID-%d): %v", m.KpiMonMetricMap[int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value)], int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value), metricValue)
				m.updateKpiMonResults(cid, plmnID, eci, m.KpiMonMetricMap[int(indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasId().Value)], metricValue, tmpTimestamp)
			} else {
				log.Debugf("Value in Indication message for type %v: %v", indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue(), metricValue)
				m.updateKpiMonResults(cid, plmnID, eci, indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()[j].GetMeasType().GetMeasName().GetValue(), metricValue, tmpTimestamp)
			}
		}
	}
	m.KpiMonMutex.Unlock()
}

func (m *Monitor) getCellIdentitiesFromHeader(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) (string, string, error) {
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

func (m *Monitor) getTimeStampFromHeader(header *e2sm_kpm_v2.E2SmKpmIndicationHeaderFormat1) uint64 {
	timeBytes := (*header).GetColletStartTime().Value
	timeInt32 := binary.BigEndian.Uint32(timeBytes)
	return uint64(timeInt32)
}
