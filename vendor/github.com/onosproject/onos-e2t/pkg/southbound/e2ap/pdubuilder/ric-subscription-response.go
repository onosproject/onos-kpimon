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

const mask20bitricResponse = 0xFFFFF

func CreateRicSubscriptionResponseE2apPdu(
	ricReq *types.RicRequest, ranFuncID types.RanFunctionID, ricActionsAccepted []*types.RicActionID) (
	*e2appdudescriptions.E2ApPdu, error) {

	if ricReq.RequestorID|mask20bitricResponse > mask20bitricResponse {
		return nil, fmt.Errorf("expecting 20 bit identifier for RIC. Got %0x", ricReq.RequestorID)
	}
	if ricReq.InstanceID|mask20bitricResponse > mask20bitricResponse {
		return nil, fmt.Errorf("expecting 20 bit identifier for RIC. Got %0x", ricReq.InstanceID)
	}

	ricRequestID := e2appducontents.RicsubscriptionResponseIes_RicsubscriptionResponseIes29{
		Id:          int32(v1beta1.ProtocolIeIDRicrequestID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicrequestId{
			RicRequestorId: int32(ricReq.RequestorID), // sequence from e2ap-v01.00.asn1:1126
			RicInstanceId:  int32(ricReq.InstanceID),  // sequence from e2ap-v01.00.asn1:1127
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ranFunctionID := e2appducontents.RicsubscriptionResponseIes_RicsubscriptionResponseIes5{
		Id:          int32(v1beta1.ProtocolIeIDRanfunctionID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RanfunctionId{
			Value: int32(ranFuncID), // range of Integer from e2ap-v01.00.asn1:1050, value from line 1277
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricActionAdmit := e2appducontents.RicsubscriptionResponseIes_RicsubscriptionResponseIes17{
		Id:          int32(v1beta1.ProtocolIeIDRicactionsAdmitted),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2appducontents.RicactionAdmittedList{
			Value: make([]*e2appducontents.RicactionAdmittedItemIes, 0),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	for _, raa := range ricActionsAccepted {
		raaIe := &e2appducontents.RicactionAdmittedItemIes{
			Id:          int32(v1beta1.ProtocolIeIDRicactionAdmittedItem),
			Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE),
			Value: &e2appducontents.RicactionAdmittedItem{
				RicActionId: &e2apies.RicactionId{
					Value: int32(*raa),
				},
			},
			Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
		}
		ricActionAdmit.GetValue().Value = append(ricActionAdmit.GetValue().Value, raaIe)
	}

	ricActionNotAdmit := e2appducontents.RicsubscriptionResponseIes_RicsubscriptionResponseIes18{
		Id:          int32(v1beta1.ProtocolIeIDRicactionsNotAdmitted),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2appducontents.RicactionNotAdmittedList{
			Value: make([]*e2appducontents.RicactionNotAdmittedItemIes, 0),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_OPTIONAL),
	}
	// TODO: Add in generation of rejected actions

	e2apPdu := e2appdudescriptions.E2ApPdu{
		E2ApPdu: &e2appdudescriptions.E2ApPdu_SuccessfulOutcome{
			SuccessfulOutcome: &e2appdudescriptions.SuccessfulOutcome{
				ProcedureCode: &e2appdudescriptions.E2ApElementaryProcedures{
					RicSubscription: &e2appdudescriptions.RicSubscription{
						SuccessfulOutcome: &e2appducontents.RicsubscriptionResponse{
							ProtocolIes: &e2appducontents.RicsubscriptionResponseIes{
								E2ApProtocolIes29: &ricRequestID,      //RIC request ID
								E2ApProtocolIes5:  &ranFunctionID,     //RAN function ID
								E2ApProtocolIes17: &ricActionAdmit,    // RIC action Admitted items ---> EMPTY !!
								E2ApProtocolIes18: &ricActionNotAdmit, // RIC action not Admitted items --> EMPTY !!
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
