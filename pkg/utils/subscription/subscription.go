// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package subscription

import (
	"github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	subapi "github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/pdubuilder"
	e2smkpmv2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	"google.golang.org/protobuf/proto"
)

// CreateSubscriptionActions creates subscription actions
func CreateSubscriptionActions(measurements []*topoapi.KPMMeasurement, cells []*topoapi.E2Cell, granularity uint32, subID int64) ([]subscription.Action, error) {
	actions := make([]subscription.Action, 0)

	for index, cell := range cells {
		measInfoList := &e2smkpmv2.MeasurementInfoList{
			Value: make([]*e2smkpmv2.MeasurementInfoItem, 0),
		}
		for _, measurement := range measurements {
			measTypeMeasName, err := pdubuilder.CreateMeasurementTypeMeasName(measurement.GetName())
			if err != nil {
				return nil, err
			}

			meanInfoItem, err := pdubuilder.CreateMeasurementInfoItem(measTypeMeasName, nil)
			if err != nil {
				return nil, err
			}
			measInfoList.Value = append(measInfoList.Value, meanInfoItem)

			actionDefinition, err := pdubuilder.CreateActionDefinitionFormat1(cell.GetCID(), measInfoList, granularity, subID)
			if err != nil {
				return nil, err
			}

			// TODO ric style types should be retrieved from R-NIB
			e2smKpmActionDefinition, err := pdubuilder.CreateE2SmKpmActionDefinitionFormat1(3, actionDefinition)
			if err != nil {
				return nil, err
			}

			e2smKpmActionDefinitionProto, err := proto.Marshal(e2smKpmActionDefinition)
			if err != nil {
				return nil, err
			}

			action := &subscription.Action{
				ID:   int32(index),
				Type: subscription.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &subscription.SubsequentAction{
					Type:       subscription.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: subscription.TimeToWait_TIME_TO_WAIT_ZERO,
				},
				Payload: subscription.Payload{
					Encoding: subscription.Encoding_ENCODING_PROTO,
					Data:     e2smKpmActionDefinitionProto,
				},
			}

			actions = append(actions, *action)
		}

	}
	return actions, nil
}

// CreateEventTriggerData creates event trigger data
func CreateEventTriggerData(rtPeriod uint32) ([]byte, error) {
	e2SmKpmEventTriggerDefinition, err := pdubuilder.CreateE2SmKpmEventTriggerDefinition(rtPeriod)
	if err != nil {
		return []byte{}, err
	}

	err = e2SmKpmEventTriggerDefinition.Validate()
	if err != nil {
		return []byte{}, err
	}

	protoBytes, err := proto.Marshal(e2SmKpmEventTriggerDefinition)
	if err != nil {
		return []byte{}, err
	}

	return protoBytes, nil
}

// SubRequest subscription request information
type SubRequest struct {
	NodeID              string
	ServiceModelName    subapi.ServiceModelName
	ServiceModelVersion subapi.ServiceModelVersion
	Actions             []subapi.Action
	EncodingType        subapi.Encoding
	EventTrigger        []byte
}

// Create  creates a subscription request
func (subRequest *SubRequest) Create() (subapi.SubscriptionDetails, error) {
	subReq := subapi.SubscriptionDetails{
		E2NodeID: subapi.E2NodeID(subRequest.NodeID),
		ServiceModel: subapi.ServiceModel{
			Name:    subRequest.ServiceModelName,
			Version: subRequest.ServiceModelVersion,
		},
		EventTrigger: subapi.EventTrigger{
			Payload: subapi.Payload{
				Encoding: subRequest.EncodingType,
				Data:     subRequest.EventTrigger,
			},
		},
		Actions: subRequest.Actions,
	}

	return subReq, nil

}
