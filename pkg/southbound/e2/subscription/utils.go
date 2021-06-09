// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package subscription

import (
	"context"

	"github.com/google/uuid"
	e2api "github.com/onosproject/onos-api/go/onos/e2t/e2/v1beta1"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/pdubuilder"
	e2smkpmv2 "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/v2/e2sm-kpm-v2"
	actionsstore "github.com/onosproject/onos-kpimon/pkg/store/actions"
	"google.golang.org/protobuf/proto"
)

// createSubscriptionActions creates subscription actions
func (m *Manager) createSubscriptionActions(ctx context.Context, measurements []*topoapi.KPMMeasurement, cells []*topoapi.E2Cell, granularity uint32) ([]e2api.Action, error) {
	actions := make([]e2api.Action, 0)

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

			subID := int64(uuid.New().ID())
			actionDefinition, err := pdubuilder.CreateActionDefinitionFormat1(cell.GetCID(), measInfoList, granularity, subID)
			if err != nil {
				return nil, err
			}

			key := actionsstore.NewKey(actionsstore.SubscriptionID{
				SubID: subID,
			})
			// TODO clean up this store if we delete subscriptions
			_, err = m.actionStore.Put(ctx, key, actionDefinition)
			if err != nil {
				log.Warn(err)
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

			action := &e2api.Action{
				ID:   int32(index),
				Type: e2api.ActionType_ACTION_TYPE_REPORT,
				SubsequentAction: &e2api.SubsequentAction{
					Type:       e2api.SubsequentActionType_SUBSEQUENT_ACTION_TYPE_CONTINUE,
					TimeToWait: e2api.TimeToWait_TIME_TO_WAIT_ZERO,
				},
				Payload: e2smKpmActionDefinitionProto,
			}

			actions = append(actions, *action)
		}

	}
	return actions, nil
}
