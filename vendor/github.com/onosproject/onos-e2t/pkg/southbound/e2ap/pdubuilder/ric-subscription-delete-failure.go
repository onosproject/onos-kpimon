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

func CreateRicSubscriptionDeleteFailureE2apPdu(
	ricReq *types.RicRequest, ranFuncID types.RanFunctionID, cause *e2apies.Cause,
	failureProcCode v1beta1.ProcedureCodeT, failureCrit e2ap_commondatatypes.Criticality,
	failureTrigMsg e2ap_commondatatypes.TriggeringMessage, reqID *types.RicRequest,
	critDiags []*types.CritDiag) (
	*e2appdudescriptions.E2ApPdu, error) {

	ricRequestID := e2appducontents.RicsubscriptionDeleteFailureIes_RicsubscriptionDeleteFailureIes29{
		Id:          int32(v1beta1.ProtocolIeIDRicrequestID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicrequestId{
			RicRequestorId: int32(ricReq.RequestorID), // sequence from e2ap-v01.00.asn1:1126
			RicInstanceId:  int32(ricReq.InstanceID),  // sequence from e2ap-v01.00.asn1:1127
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ranFunctionID := e2appducontents.RicsubscriptionDeleteFailureIes_RicsubscriptionDeleteFailureIes5{
		Id:          int32(v1beta1.ProtocolIeIDRanfunctionID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RanfunctionId{
			Value: int32(ranFuncID), // range of Integer from e2ap-v01.00.asn1:1050, value from line 1277
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	causeOfFailure := e2appducontents.RicsubscriptionDeleteFailureIes_RicsubscriptionDeleteFailureIes1{
		Id:          int32(v1beta1.ProtocolIeIDCause),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE),
		Value:       cause,
		Presence:    int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	criticalityDiagnostics := e2appducontents.RicsubscriptionDeleteFailureIes_RicsubscriptionDeleteFailureIes2{
		Id:          int32(v1beta1.ProtocolIeIDCriticalityDiagnostics),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE),
		Value: &e2apies.CriticalityDiagnostics{
			ProcedureCode: &e2ap_commondatatypes.ProcedureCode{
				Value: int32(failureProcCode), // range of Integer from e2ap-v01.00.asn1:1206, value were taken from line 1236 (same file)
			},
			TriggeringMessage:    failureTrigMsg,
			ProcedureCriticality: failureCrit, // from e2ap-v01.00.asn1:153
			RicRequestorId: &e2apies.RicrequestId{
				RicRequestorId: int32(reqID.RequestorID),
				RicInstanceId:  int32(reqID.InstanceID),
			},
			IEsCriticalityDiagnostics: &e2apies.CriticalityDiagnosticsIeList{
				Value: make([]*e2apies.CriticalityDiagnosticsIeItem, 0),
			},
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_OPTIONAL),
	}

	for _, critDiag := range critDiags {
		criticDiagnostics := e2apies.CriticalityDiagnosticsIeItem{
			IEcriticality: critDiag.IECriticality,
			IEId: &e2ap_commondatatypes.ProtocolIeId{
				Value: int32(critDiag.IEId), // value were taken from e2ap-v01.00.asn1:1278
			},
			TypeOfError: critDiag.TypeOfError,
		}
		criticalityDiagnostics.Value.IEsCriticalityDiagnostics.Value = append(criticalityDiagnostics.Value.IEsCriticalityDiagnostics.Value, &criticDiagnostics)
	}

	e2apPdu := e2appdudescriptions.E2ApPdu{
		E2ApPdu: &e2appdudescriptions.E2ApPdu_UnsuccessfulOutcome{
			UnsuccessfulOutcome: &e2appdudescriptions.UnsuccessfulOutcome{
				ProcedureCode: &e2appdudescriptions.E2ApElementaryProcedures{
					RicSubscriptionDelete: &e2appdudescriptions.RicSubscriptionDelete{
						UnsuccessfulOutcome: &e2appducontents.RicsubscriptionDeleteFailure{
							ProtocolIes: &e2appducontents.RicsubscriptionDeleteFailureIes{
								E2ApProtocolIes29: &ricRequestID,  //RIC request ID
								E2ApProtocolIes5:  &ranFunctionID, //RAN function ID
								E2ApProtocolIes1:  &causeOfFailure,
								E2ApProtocolIes2:  &criticalityDiagnostics,
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
