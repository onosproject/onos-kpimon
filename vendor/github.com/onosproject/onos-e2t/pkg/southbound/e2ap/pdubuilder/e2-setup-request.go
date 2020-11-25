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

func CreateE2SetupRequestPdu(plmnID types.PlmnID, ranFunctionIds types.RanFunctions) (*e2appdudescriptions.E2ApPdu, error) {

	gnbIDIe := e2appducontents.E2SetupRequestIes_E2SetupRequestIes3{
		Id:          int32(v1beta1.ProtocolIeIDGlobalE2nodeID),
		Presence:    int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2apies.GlobalE2NodeId{
			GlobalE2NodeId: &e2apies.GlobalE2NodeId_GNb{
				GNb: &e2apies.GlobalE2NodeGnbId{
					GlobalGNbId: &e2apies.GlobalgNbId{
						PlmnId: &e2ap_commondatatypes.PlmnIdentity{
							Value: []byte{plmnID[0], plmnID[1], plmnID[2]},
						},
						GnbId: &e2apies.GnbIdChoice{
							GnbIdChoice: &e2apies.GnbIdChoice_GnbId{
								GnbId: &e2ap_commondatatypes.BitString{
									Value: 0x9bcd4,
									Len:   22,
								}},
						},
					},
				},
			},
		},
	}

	ranFunctions := e2appducontents.E2SetupRequestIes_E2SetupRequestIes10{
		Id:          int32(v1beta1.ProtocolIeIDRanfunctionsAdded),
		Presence:    int32(e2ap_commondatatypes.Presence_PRESENCE_OPTIONAL),
		Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_REJECT),
		Value: &e2appducontents.RanfunctionsList{
			Value: make([]*e2appducontents.RanfunctionItemIes, 0),
		},
	}

	for id, ranFunctionID := range ranFunctionIds {
		ranFunction := e2appducontents.RanfunctionItemIes{
			E2ApProtocolIes10: &e2appducontents.RanfunctionItemIes_RanfunctionItemIes8{
				Id:          int32(v1beta1.ProtocolIeIDRanfunctionItem),
				Presence:    int32(e2ap_commondatatypes.Presence_PRESENCE_MANDATORY),
				Criticality: int32(e2ap_commondatatypes.Criticality_CRITICALITY_IGNORE),
				Value: &e2appducontents.RanfunctionItem{
					RanFunctionId: &e2apies.RanfunctionId{
						Value: int32(id),
					},
					RanFunctionDefinition: &e2ap_commondatatypes.RanfunctionDefinition{
						Value: []byte(ranFunctionID.Description),
					},
					RanFunctionRevision: &e2apies.RanfunctionRevision{
						Value: int32(ranFunctionID.Revision),
					},
				},
			},
		}
		ranFunctions.Value.Value = append(ranFunctions.Value.Value, &ranFunction)
	}

	e2apPdu := e2appdudescriptions.E2ApPdu{
		E2ApPdu: &e2appdudescriptions.E2ApPdu_InitiatingMessage{
			InitiatingMessage: &e2appdudescriptions.InitiatingMessage{
				ProcedureCode: &e2appdudescriptions.E2ApElementaryProcedures{
					E2Setup: &e2appdudescriptions.E2Setup{
						InitiatingMessage: &e2appducontents.E2SetupRequest{
							ProtocolIes: &e2appducontents.E2SetupRequestIes{
								E2ApProtocolIes3:  &gnbIDIe,
								E2ApProtocolIes10: &ranFunctions,
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
