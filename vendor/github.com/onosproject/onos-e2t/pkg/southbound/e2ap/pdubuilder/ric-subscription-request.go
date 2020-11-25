// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0
package pdubuilder

import (
	"fmt"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1"
	e2ap_commondatatypes "github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2ap-commondatatypes"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2apies"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2appducontents"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2appdudescriptions"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
)

func CreateRicSubscriptionRequestE2apPdu(ricReq types.RicRequest,
	ranFuncID types.RanFunctionID, ricEventDef types.RicEventDefintion,
	ricActionsToBeSetup map[types.RicActionID]types.RicActionDef) (
	*e2appdudescriptions.E2ApPdu, error) {

	ricRequestID := e2appducontents.RicsubscriptionRequestIes_RicsubscriptionRequestIes29{
		Id:          int32(v1beta1.ProtocolIeIDRicrequestID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicrequestId{
			RicRequestorId: int32(ricReq.RequestorID), // sequence from e2ap-v01.00.asn1:1126
			RicInstanceId:  int32(ricReq.InstanceID),  // sequence from e2ap-v01.00.asn1:1127
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ranFunctionID := e2appducontents.RicsubscriptionRequestIes_RicsubscriptionRequestIes5{
		Id:          int32(v1beta1.ProtocolIeIDRanfunctionID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RanfunctionId{
			Value: int32(ranFuncID), // range of Integer from e2ap-v01.00.asn1:1050, value from line 1277
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricSubscriptionDetails := e2appducontents.RicsubscriptionRequestIes_RicsubscriptionRequestIes30{
		Id:          int32(v1beta1.ProtocolIeIDRicsubscriptionDetails),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2appducontents.RicsubscriptionDetails{
			RicEventTriggerDefinition: &e2ap_commondatatypes.RiceventTriggerDefinition{},
			RicActionToBeSetupList: &e2appducontents.RicactionsToBeSetupList{
				Value: make([]*e2appducontents.RicactionToBeSetupItemIes, 0),
			},
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}
	ricSubscriptionDetails.Value.RicEventTriggerDefinition.Value = make([]byte, len(ricEventDef))
	copy(ricSubscriptionDetails.Value.RicEventTriggerDefinition.Value, ricEventDef)
	// ricEventDef value taken from e2ap-v01.00.asn1:1297

	for ricActionID, ricAction := range ricActionsToBeSetup {
		ricActionToSetup := e2appducontents.RicactionToBeSetupItemIes{
			Id:          int32(v1beta1.ProtocolIeIDRicactionToBeSetupItem),
			Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE),
			Value: &e2appducontents.RicactionToBeSetupItem{
				RicActionId: &e2apies.RicactionId{
					Value: int32(ricActionID), // range of Integer from e2ap-v01.00.asn1:1059, value from line 1283
				},
				RicActionType:       ricAction.RicActionType,
				RicActionDefinition: &e2ap_commondatatypes.RicactionDefinition{},
				RicSubsequentAction: &e2apies.RicsubsequentAction{
					RicSubsequentActionType: ricAction.RicSubsequentAction,
					RicTimeToWait:           ricAction.Ricttw,
				},
			},
			Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
		}
		ricActionToSetup.Value.RicActionDefinition.Value = make([]byte, len(ricAction.RicActionDefinition))
		copy(ricActionToSetup.Value.RicActionDefinition.Value, ricAction.RicActionDefinition)
		//ricEventDef value taken from e2ap-v01.00.asn1:1285
		ricSubscriptionDetails.Value.RicActionToBeSetupList.Value = append(ricSubscriptionDetails.Value.RicActionToBeSetupList.Value, &ricActionToSetup)
	}

	e2apPdu := e2appdudescriptions.E2ApPdu{
		E2ApPdu: &e2appdudescriptions.E2ApPdu_InitiatingMessage{
			InitiatingMessage: &e2appdudescriptions.InitiatingMessage{
				ProcedureCode: &e2appdudescriptions.E2ApElementaryProcedures{
					RicSubscription: &e2appdudescriptions.RicSubscription{
						InitiatingMessage: &e2appducontents.RicsubscriptionRequest{
							ProtocolIes: &e2appducontents.RicsubscriptionRequestIes{
								E2ApProtocolIes29: &ricRequestID,           // RIC request ID
								E2ApProtocolIes5:  &ranFunctionID,          // RAN function ID
								E2ApProtocolIes30: &ricSubscriptionDetails, // RIC subscription details
							},
						},
					},
				},
			},
		},
	}
	if err := e2apPdu.Validate(); err != nil {
		return nil, fmt.Errorf("error validating E2ApPDU %s", err.Error())
	}
	return &e2apPdu, nil
}
