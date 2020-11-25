// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0
package pdubuilder

import (
	"fmt"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1"
	e2ap_commondatatypes "github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2ap-commondatatypes"
	e2ap_constants "github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2ap-constants"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2apies"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2appducontents"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2appdudescriptions"
	"github.com/onosproject/onos-e2t/pkg/southbound/e2ap/types"
)

func RicIndicationE2apPdu(ricReqID types.RicRequest, ranFuncID types.RanFunctionID,
	ricAction e2apies.RicactionType, ricSn types.RicIndicationSn, ricIndicationType e2apies.RicindicationType,
	ricIndHd types.RicIndicationHeader, ricIndMsg types.RicIndicationMessage, ricCallPrID types.RicCallProcessID) (
	*e2appdudescriptions.E2ApPdu, error) {

	ricRequestID := e2appducontents.RicindicationIes_RicindicationIes29{
		Id:          int32(v1beta1.ProtocolIeIDRicrequestID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicrequestId{
			RicRequestorId: int32(ricReqID.RequestorID), // sequence from e2ap-v01.00.asn1:1126
			RicInstanceId:  int32(ricReqID.InstanceID),  // sequence from e2ap-v01.00.asn1:1127
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ranFunctionID := e2appducontents.RicindicationIes_RicindicationIes5{
		Id:          int32(v1beta1.ProtocolIeIDRanfunctionID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RanfunctionId{
			Value: int32(ranFuncID), // range of Integer from e2ap-v01.00.asn1:1050, value from line 1277
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricAct := e2appducontents.RicindicationIes_RicindicationIes15{
		Id:          int32(v1beta1.ProtocolIeIDRicactionID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicactionId{
			Value: int32(ricAction),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricIndicationSn := e2appducontents.RicindicationIes_RicindicationIes27{
		Id:          int32(v1beta1.ProtocolIeIDRicindicationSn),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.RicindicationSn{
			Value: int32(ricSn),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_OPTIONAL),
	}

	ricIndType := e2appducontents.RicindicationIes_RicindicationIes28{
		Id:          int32(v1beta1.ProtocolIeIDRicindicationType),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value:       ricIndicationType,
		Presence:    int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricIndHeader := e2appducontents.RicindicationIes_RicindicationIes25{
		Id:          int32(v1beta1.ProtocolIeIDRicindicationHeader),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2ap_commondatatypes.RicindicationHeader{
			Value: []byte(ricIndHd),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricIndMessage := e2appducontents.RicindicationIes_RicindicationIes26{
		Id:          int32(v1beta1.ProtocolIeIDRicindicationMessage),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2ap_commondatatypes.RicindicationMessage{
			Value: []byte(ricIndMsg),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
	}

	ricCallProcessID := e2appducontents.RicindicationIes_RicindicationIes20{
		Id:          int32(v1beta1.ProtocolIeIDRiccallProcessID),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2ap_commondatatypes.RiccallProcessId{
			Value: []byte(ricCallPrID),
		},
		Presence: int32(e2ap_commondatatypes.Presence_PRESENCE_OPTIONAL),
	}

	e2apPdu := e2appdudescriptions.E2ApPdu{
		E2ApPdu: &e2appdudescriptions.E2ApPdu_InitiatingMessage{
			InitiatingMessage: &e2appdudescriptions.InitiatingMessage{
				ProcedureCode: &e2appdudescriptions.E2ApElementaryProcedures{
					RicIndication: &e2appdudescriptions.RicIndication{
						InitiatingMessage: &e2appducontents.Ricindication{
							ProtocolIes: &e2appducontents.RicindicationIes{
								E2ApProtocolIes29: &ricRequestID,     // RIC Requestor & RIC Instance ID
								E2ApProtocolIes5:  &ranFunctionID,    // RAN function ID
								E2ApProtocolIes15: &ricAct,           // RIC Action
								E2ApProtocolIes27: &ricIndicationSn,  // RIC Indication Sn (Sequence Number?)
								E2ApProtocolIes28: &ricIndType,       // RIC Indication Type
								E2ApProtocolIes25: &ricIndHeader,     // RIC Indication Header
								E2ApProtocolIes26: &ricIndMessage,    // RIC Indication Message
								E2ApProtocolIes20: &ricCallProcessID, // RIC Call Process ID
							},
						},
						ProcedureCode: &e2ap_constants.IdRicindication{
							Value: int32(v1beta1.ProcedureCodeIDRICindication),
						},
						Criticality: &e2ap_commondatatypes.CriticalityIgnore{
							Criticality: e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE,
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
