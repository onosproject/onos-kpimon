// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package e2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	appConfig "github.com/onosproject/onos-kpimon/pkg/config"

	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/pdubuilder"
	e2sm_kpm_v2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	e2client "github.com/onosproject/onos-ric-sdk-go/pkg/e2"
	"google.golang.org/protobuf/proto"

	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"

	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	subuitls "github.com/onosproject/onos-kpimon/pkg/utils/subscription"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/config/event"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	sdkSub "github.com/onosproject/onos-ric-sdk-go/pkg/e2/subscription"
)

var log = logging.GetLogger("southbound", "ricapie2")

// KpmServiceModelOIDV2 is the OID for KPM V2.0
const KpmServiceModelOIDV2 = "1.3.6.1.4.1.53148.1.2.2.2"

// NewE2Client generates a new E2 client
func NewE2Client(e2tEndpoint string, e2subEndpoint string, ricActionID int32, smName string, smVersion string, kpiMonMetricMap map[int]string, appConfig *appConfig.AppConfig) *E2Session {
	e2Session := &E2Session{
		E2SubEndpoint:   e2subEndpoint,
		E2TEndpoint:     e2tEndpoint,
		E2SubInstances:  make(map[string]sdkSub.Context),
		RicActionID:     types.RicActionID(ricActionID),
		SMName:          smName,
		SMVersion:       smVersion,
		KpiMonMetricMap: kpiMonMetricMap,
		SubDelTriggers:  make(map[string]chan bool),
		appConfig:       appConfig,
	}

	return e2Session
}

// E2Client is an interface for interaction with an E2 node via E2T
type E2Client interface {
	Run(chan indication.Indication, admin.E2AdminSession)
}

// E2Session is an abstract struct of E2 session
type E2Session struct {
	E2SubEndpoint   string
	E2SubInstances  map[string]sdkSub.Context
	SubDelTriggers  map[string]chan bool
	E2TEndpoint     string
	RicActionID     types.RicActionID
	appConfig       *appConfig.AppConfig
	EventMutex      sync.RWMutex
	ConfigEventCh   chan event.Event
	SMName          string
	SMVersion       string
	KpiMonMetricMap map[int]string
	mu              sync.RWMutex
	subID           int64
}

func (s *E2Session) processConfigEvents() {
	for configEvent := range s.ConfigEventCh {
		if configEvent.Key == utils.ReportPeriodConfigPath {
			log.Infof("Report Period: Config Event received: %v", configEvent)
			err := s.deleteE2Subscriptions()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *E2Session) watchConfigChanges() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.appConfig.Watch(ctx, s.ConfigEventCh)
	if err != nil {
		return err
	}

	s.processConfigEvents()
	return nil
}

func (s *E2Session) deleteE2Subscriptions() error {
	for k := range s.E2SubInstances {
		err := s.deleteE2Subscription(k)
		if err != nil {
			log.Errorf("Failed to delete E2 subscription: %v", err)
			return err
		}
		s.SubDelTriggers[k] <- true
	}
	return nil
}

func (s *E2Session) deleteE2Subscription(e2NodeID string) error {
	log.Infof("Start deleting subscription - E2NodeID: %v", e2NodeID)
	err := s.E2SubInstances[e2NodeID].Close()
	return err
}

// Run starts E2 session
func (s *E2Session) Run(indChan chan indication.Indication, adminSession admin.E2AdminSession) {
	log.Info("Started KPIMON Southbound session")
	s.ConfigEventCh = make(chan event.Event)
	go func() {
		_ = s.watchConfigChanges()
	}()
	s.manageConnections(indChan, adminSession)
}

func (s *E2Session) manageConnections(indChan chan indication.Indication, adminSession admin.E2AdminSession) {
	for {
		nodeIDs, err := adminSession.GetListE2NodeIDs()
		if err != nil {
			log.Errorf("Cannot get NodeIDs through Admin API: %s", err)
			continue
		} else if len(nodeIDs) == 0 {
			log.Warn("CU-CP is not running - wait until CU-CP is ready")
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		log.Infof("Received E2Nodes: %v", nodeIDs)

		hasKpiMonMetricMap := true
		for _, id := range nodeIDs {
			ranFuncDesc, err := s.getRanFuncDesc(id, adminSession)
			if err != nil {
				hasKpiMonMetricMap = false
				break
			}
			for range ranFuncDesc.GetRicKpmNodeList()[0].GetCellMeasurementObjectList() {
				for _, measInfoActionItem := range ranFuncDesc.GetRicReportStyleList()[0].GetMeasInfoActionList().GetValue() {
					actionName := measInfoActionItem.GetMeasName()
					actionID := measInfoActionItem.GetMeasId()
					log.Debugf("Check RAN function description to make KpiMonMetricMap - ranFuncDesc: %v", ranFuncDesc)
					log.Debugf("Check MeasInfoActionItem to make KpiMonMetricMap - ranFuncDesc: %v", measInfoActionItem)
					log.Debugf("KpiMonMetricMap generation - name:%v, id:%d", actionName, actionID)
					s.KpiMonMetricMap[int(actionID.Value)] = actionName.Value
				}
			}
		}

		log.Debugf("KPIMONMetricMap: %v", s.KpiMonMetricMap)

		if !hasKpiMonMetricMap {
			continue
		}

		var wg sync.WaitGroup
		for _, id := range nodeIDs {
			wg.Add(1)
			if _, ok := s.SubDelTriggers[id]; !ok {
				s.SubDelTriggers[id] = make(chan bool)
			}
			go func(id string, wg *sync.WaitGroup) {
				defer wg.Done()
				for {
					ranFuncDesc, err := s.getRanFuncDesc(id, adminSession)
					if err != nil {
						log.Error(err)
						time.Sleep(1 * time.Second)
					}
					s.manageConnection(indChan, id, ranFuncDesc)
				}
			}(id, &wg)
		}
		wg.Wait()
	}
}

func (s *E2Session) getRanFuncDesc(nodeID string, adminSession admin.E2AdminSession) (*e2sm_kpm_v2.E2SmKpmRanfunctionDescription, error) {
	ranFunctions, err := adminSession.GetRANFunctions(nodeID)
	if err != nil {
		return nil, err
	}

	ranFunctionDesc := &e2sm_kpm_v2.E2SmKpmRanfunctionDescription{}
	ranFunctionFound := false
	for _, ranFunction := range ranFunctions {
		if ranFunction.Oid == KpmServiceModelOIDV2 {
			err = proto.Unmarshal(ranFunction.Description, ranFunctionDesc)
			if err != nil {
				return nil, err
			}
			ranFunctionFound = true
		}
	}
	if !ranFunctionFound {
		return nil, fmt.Errorf("cannot find RANFunction OID (%s) from nodeID %s", KpmServiceModelOIDV2, nodeID)
	}

	log.Debugf("RANFunctionDesc: %v", ranFunctionDesc)
	return ranFunctionDesc, nil
}

func (s *E2Session) getCellObjectID(cellMeasureObjectItem *e2sm_kpm_v2.CellMeasurementObjectItem) string {
	return cellMeasureObjectItem.CellObjectId.Value
}

func (s *E2Session) createActionDefinition(ranFuncDesc *e2sm_kpm_v2.E2SmKpmRanfunctionDescription) (map[string]*e2sm_kpm_v2.E2SmKpmActionDefinition, error) {
	result := make(map[string]*e2sm_kpm_v2.E2SmKpmActionDefinition)
	granularity, err := s.appConfig.GetGranularityPeriod()
	if err != nil {
		return nil, err
	}
	for _, cellMeasObjItem := range ranFuncDesc.GetRicKpmNodeList()[0].GetCellMeasurementObjectList() {
		cellObjID := s.getCellObjectID(cellMeasObjItem)
		measInfoList := &e2sm_kpm_v2.MeasurementInfoList{
			Value: make([]*e2sm_kpm_v2.MeasurementInfoItem, 0),
		}
		for _, measInfoActionItem := range ranFuncDesc.GetRicReportStyleList()[0].GetMeasInfoActionList().GetValue() {
			// for test with name
			actionName := measInfoActionItem.GetMeasName()
			tmpMeasTypeMeasName, err := pdubuilder.CreateMeasurementTypeMeasName(actionName.Value)
			if err != nil {
				return nil, err
			}

			tmpMeasInfoItem1, err := pdubuilder.CreateMeasurementInfoItem(tmpMeasTypeMeasName, nil)
			if err != nil {
				return nil, err
			}
			measInfoList.Value = append(measInfoList.Value, tmpMeasInfoItem1)

			// for test with ID
			//actionID := measInfoActionItem.GetMeasId()
			//tmpMeasTypeMeasID, err := pdubuilder.CreateMeasurementTypeMeasID(actionID.Value)
			//if err != nil {
			//	return nil, err
			//}
			//tmpMeasInfoItem2, err := pdubuilder.CreateMeasurementInfoItem(tmpMeasTypeMeasID, nil)
			//if err != nil {
			//	return nil, err
			//}
			//measInfoList.Value = append(measInfoList.Value, tmpMeasInfoItem2)
		}

		// Generate subscription ID - started from 1 to maximum int64
		s.mu.Lock()
		s.subID++
		subID := s.subID
		s.mu.Unlock()

		log.Debugf("subID for %v: %v", cellObjID, subID)

		actionDefinitionCell, err := pdubuilder.CreateActionDefinitionFormat1(cellObjID, measInfoList, uint32(granularity), 10)
		if err != nil {
			return nil, err
		}

		e2smKpmActionDefinitionCell, err := pdubuilder.CreateE2SmKpmActionDefinitionFormat1(1, actionDefinitionCell)
		if err != nil {
			return nil, err
		}

		result[cellObjID] = e2smKpmActionDefinitionCell
	}
	log.Debugf("ActionDefinitions: %v", result)
	return result, nil
}

func (s *E2Session) manageConnection(indChan chan indication.Indication, nodeID string, ranFuncDesc *e2sm_kpm_v2.E2SmKpmRanfunctionDescription) {
	err := s.createE2Subscription(indChan, nodeID, ranFuncDesc)
	if err != nil {
		log.Errorf("Error happens when creating E2 subscription - %s", err)
	}
}

func (s *E2Session) createSubscriptionRequestWithActionDefinition(nodeID string, ranFuncDesc *e2sm_kpm_v2.E2SmKpmRanfunctionDescription) (subscription.SubscriptionDetails, error) {
	actionDefMap, err := s.createActionDefinition(ranFuncDesc)
	if err != nil {
		return subscription.SubscriptionDetails{}, err
	}
	actionDefinition, err := s.createSubscriptionActionsList(actionDefMap)
	if err != nil {
		return subscription.SubscriptionDetails{}, err
	}

	rtPeriod, err := s.appConfig.GetGranularityPeriod()
	if err != nil {
		return subscription.SubscriptionDetails{}, err
	}

	eventTriggerData, err := subuitls.CreateEventTriggerData(uint32(rtPeriod))
	if err != nil {
		return subscription.SubscriptionDetails{}, err
	}

	sub := subscription.SubscriptionDetails{
		E2NodeID: subscription.E2NodeID(nodeID),
		ServiceModel: subscription.ServiceModel{
			Name:    subscription.ServiceModelName(s.SMName),
			Version: subscription.ServiceModelVersion(s.SMVersion),
		},
		EventTrigger: subscription.EventTrigger{
			Payload: subscription.Payload{
				Encoding: subscription.Encoding_ENCODING_PROTO,
				Data:     eventTriggerData,
			},
		},
		Actions: actionDefinition,
	}
	log.Debugf("subscription request: %v", sub)

	return sub, nil
}

func (s *E2Session) createSubscriptionActionsList(e2smKpmActionDefinitions map[string]*e2sm_kpm_v2.E2SmKpmActionDefinition) ([]subscription.Action, error) {
	result := make([]subscription.Action, 0)
	i := int32(0)
	for _, v := range e2smKpmActionDefinitions {
		protoBytesCell, err := proto.Marshal(v)
		if err != nil {
			return nil, err
		}

		tmpAction := &subscription.Action{
			ID:   int32(s.RicActionID) + i,
			Type: subscription.ActionType_ACTION_TYPE_REPORT,
			SubsequentAction: &subscription.SubsequentAction{
				Type:       subscription.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
				TimeToWait: subscription.TimeToWait_TIME_TO_WAIT_ZERO,
			},
			Payload: subscription.Payload{
				Encoding: subscription.Encoding_ENCODING_PROTO,
				Data:     protoBytesCell,
			},
		}

		result = append(result, *tmpAction)
		i++
	}
	return result, nil
}

func (s *E2Session) createE2Subscription(indChan chan indication.Indication, nodeID string, ranFuncDesc *e2sm_kpm_v2.E2SmKpmRanfunctionDescription) error {
	log.Infof("Connecting to ONOS-E2Sub...%s", s.E2SubEndpoint)

	e2SubHost := strings.Split(s.E2SubEndpoint, ":")[0]
	e2SubPort, err := strconv.Atoi(strings.Split(s.E2SubEndpoint, ":")[1])
	if err != nil {
		log.Error("onos-e2sub's port information or endpoint information is wrong.")
		return err
	}
	e2tHost := strings.Split(s.E2TEndpoint, ":")[0]
	e2tPort, err := strconv.Atoi(strings.Split(s.E2TEndpoint, ":")[1])
	if err != nil {
		log.Error("onos-e2t's port information or endpoint information is wrong.")
		return err
	}

	clientConfig := e2client.Config{
		AppID: "onos-kpimon-v2",
		E2TService: e2client.ServiceConfig{
			Host: e2tHost,
			Port: e2tPort,
		},
		SubscriptionService: e2client.ServiceConfig{
			Host: e2SubHost,
			Port: e2SubPort,
		},
	}

	client, err := e2client.NewClient(clientConfig)
	if err != nil {
		log.Error("Can't open E2Client.")
		return err
	}

	ch := make(chan indication.Indication)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subReq, err := s.createSubscriptionRequestWithActionDefinition(nodeID, ranFuncDesc)
	if err != nil {
		log.Error("Can't create SubsdcriptionRequest message")
		return err
	}

	log.Infof("Start subscribe - E2Node: %v", nodeID)
	e2subInst, err := client.Subscribe(ctx, subReq, ch)
	s.E2SubInstances[nodeID] = e2subInst

	if err != nil {
		log.Error("Can't send SubscriptionRequest message")
		return err
	}

	log.Infof("Start forwarding Indication message to KPIMON monitor - E2Node: %v", nodeID)
	for {
		select {
		case indMsg := <-ch:
			indChan <- indMsg
		case trigger := <-s.SubDelTriggers[nodeID]:
			if trigger {
				log.Info("Reset indChan to close subscription")
				return nil
			}
		}
	}
}

var _ E2Client = &E2Session{}
