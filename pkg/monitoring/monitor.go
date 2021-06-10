// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package monitoring

import (
	"context"
	"strconv"

	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	"github.com/onosproject/onos-kpimon/pkg/store/actions"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2/v1beta1"

	appConfig "github.com/onosproject/onos-kpimon/pkg/config"

	measurmentStore "github.com/onosproject/onos-kpimon/pkg/store/measurements"

	e2smkpmv2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"google.golang.org/protobuf/proto"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	"github.com/onosproject/onos-kpimon/pkg/broker"
)

var log = logging.GetLogger("monitoring")

// NewMonitor creates a new indication monitor
func NewMonitor(streams broker.Broker, appConfig *appConfig.AppConfig,
	measurementStore measurmentStore.Store, actionsStore actions.Store) *Monitor {
	return &Monitor{
		streams:          streams,
		appConfig:        appConfig,
		measurementStore: measurementStore,
		actionStore:      actionsStore,
	}
}

// Monitor indication monitor
type Monitor struct {
	streams          broker.Broker
	measurementStore measurmentStore.Store
	actionStore      actions.Store
	appConfig        *appConfig.AppConfig
}

func (m *Monitor) processIndicationFormat1(ctx context.Context, indication e2api.Indication, measurements []*topoapi.KPMMeasurement) error {
	indHeader := e2smkpmv2.E2SmKpmIndicationHeader{}
	err := proto.Unmarshal(indication.Header, &indHeader)
	if err != nil {
		log.Warn(err)
		return err
	}

	indMessage := e2smkpmv2.E2SmKpmIndicationMessage{}
	err = proto.Unmarshal(indication.Payload, &indMessage)
	if err != nil {
		log.Warn(err)
		return err
	}

	log.Debugf("Received indication header format 1 %v:", indHeader.GetIndicationHeaderFormat1())
	log.Debugf("Received indication message format 1: %v", indMessage.GetIndicationMessageFormat1())

	startTime := getTimeStampFromHeader(indHeader.GetIndicationHeaderFormat1())
	startTimeUnixNano := toUnixNano(int64(startTime))

	granularity, err := m.appConfig.GetGranularityPeriod()
	if err != nil {
		log.Warn(err)
		return err
	}

	var cid string
	if indMessage.GetIndicationMessageFormat1().GetCellObjId() == nil {
		// Use the actions store to find cell object Id based on sub ID in action definition
		key := actions.NewKey(actions.SubscriptionID{
			SubID: indMessage.GetIndicationMessageFormat1().GetSubscriptId().GetValue(),
		})

		response, err := m.actionStore.Get(ctx, key)
		if err != nil {
			return err
		}

		actionDefinition := response.Value.(*e2smkpmv2.E2SmKpmActionDefinitionFormat1)
		cid = actionDefinition.GetCellObjId().GetValue()

	} else {
		cid = indMessage.GetIndicationMessageFormat1().GetCellObjId().Value
	}

	measDataItems := indMessage.GetIndicationMessageFormat1().GetMeasData().GetValue()
	measInfoList := indMessage.GetIndicationMessageFormat1().GetMeasInfoList().GetValue()

	measItems := make([]measurmentStore.MeasurementItem, 0)
	for i, measDataItem := range measDataItems {
		meadDataRecords := measDataItem.GetMeasRecord().GetValue()
		measRecords := make([]measurmentStore.MeasurementRecord, 0)
		for j, measDataRecord := range meadDataRecords {
			var measValue interface{}
			switch val := measDataRecord.MeasurementRecordItem.(type) {
			case *e2smkpmv2.MeasurementRecordItem_Integer:
				measValue = val.Integer

			case *e2smkpmv2.MeasurementRecordItem_Real:
				measValue = val.Real

			case *e2smkpmv2.MeasurementRecordItem_NoValue:
				measValue = val.NoValue
			default:
				measValue = 0
			}

			timeStamp := uint64(startTimeUnixNano) + granularity*uint64(1000000)*uint64(i)
			if measInfoList[j].GetMeasType().GetMeasName().GetValue() != "" {
				measName := measInfoList[j].GetMeasType().GetMeasName().GetValue()
				measRecord := measurmentStore.MeasurementRecord{
					Timestamp:        timeStamp,
					MeasurementName:  measName,
					MeasurementValue: measValue,
				}
				measRecords = append(measRecords, measRecord)
			} else if measInfoList[j].GetMeasType().GetMeasId().GetValue() != 0 {
				measID := measInfoList[j].GetMeasType().GetMeasId().GetValue()
				measIDString := strconv.Itoa(int(measID))
				measName := getMeasurementName(measIDString, measurements)
				measRecord := measurmentStore.MeasurementRecord{
					Timestamp:        timeStamp,
					MeasurementName:  measName,
					MeasurementValue: measValue,
				}
				measRecords = append(measRecords, measRecord)
			}
		}

		measItem := measurmentStore.MeasurementItem{
			MeasurementRecords: measRecords,
		}
		measItems = append(measItems, measItem)

	}

	cellID := measurmentStore.CellIdentity{
		CellID: cid,
	}

	measurementKey := measurmentStore.NewKey(cellID)
	_, err = m.measurementStore.Put(ctx, measurementKey, measItems)
	if err != nil {
		log.Warn(err)
		return err
	}
	return nil
}

func (m *Monitor) processIndication(ctx context.Context, indication e2api.Indication, measurements []*topoapi.KPMMeasurement) error {
	err := m.processIndicationFormat1(ctx, indication, measurements)
	if err != nil {
		log.Warn(err)
		return err
	}

	return nil
}

// Start start monitoring of indication messages for a given subscription ID
func (m *Monitor) Start(ctx context.Context, node e2client.Node, e2sub e2api.Subscription, measurements []*topoapi.KPMMeasurement) error {
	streamReader, err := m.streams.OpenReader(node, e2sub)
	if err != nil {
		return err
	}

	for {
		indMsg, err := streamReader.Recv(ctx)
		if err != nil {
			return err
		}
		err = m.processIndication(ctx, indMsg, measurements)
		if err != nil {
			return err
		}
	}
}
