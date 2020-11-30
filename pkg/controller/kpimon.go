// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package controller

import (
	"github.com/gogo/protobuf/proto"
	e2sm_kpm_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
	"github.com/onosproject/onos-e2t/api/e2ap/v1beta1/e2appdudescriptions"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("ctrl-kpimon")

// KpiMonCtrl is the controller for the KPI monitoring
type KpiMonCtrl struct {
	IndChan chan indication.Indication
}

// NewKpiMonController creates a new KpiMonController
func NewKpiMonController(indChan chan indication.Indication) *KpiMonCtrl {
	log.Info("Start ONOS-KPIMON Application Controller")
	return &KpiMonCtrl{
		IndChan: indChan,
	}
}

// PrintMessages is the function to print all indication messages - for debugging as of now
func (c *KpiMonCtrl) PrintMessages() {
	for indMsg := range c.IndChan {

		log.Infof("Received msg: %v", indMsg)
		indMsgAsn1Bytes := indMsg.Payload.Value.([]byte)
		e2apFmtMsg := e2appdudescriptions.E2ApPdu{}
		_ = proto.Unmarshal(indMsgAsn1Bytes, &e2apFmtMsg)

		// print header information - PLMN ID
		indHeaderAsn1Bytes := e2apFmtMsg.GetInitiatingMessage().GetProcedureCode().GetRicIndication().GetInitiatingMessage().GetProtocolIes().GetE2ApProtocolIes25().GetValue().GetValue()
		log.Infof("Header bytes in ASN.1: %v", indHeaderAsn1Bytes)

		indHeaderProtoBytes, err := utils.IndicationHeaderASN1toProto(indHeaderAsn1Bytes)
		if err != nil {
			log.Errorf("Error - converting asn1 bytes to proto bytes: %s", err)
		}

		indHeader := e2sm_kpm_ies.E2SmKpmIndicationHeader{}
		err = proto.Unmarshal(indHeaderProtoBytes, &indHeader)
		if err != nil {
			log.Errorf("Error - unmashal protobytes to struct: %s", err)
		}
		log.Infof("PLMNID: %v", indHeader.GetIndicationHeaderFormat1().GetPLmnIdentity().GetValue())

		// print numActiveUEs information in payload
		indMessageAsn1Bytes := e2apFmtMsg.GetInitiatingMessage().GetProcedureCode().GetRicIndication().GetInitiatingMessage().GetProtocolIes().GetE2ApProtocolIes26().GetValue().GetValue()

		indMessageProtoBytes, err := utils.IndicationMessageASN1toProto(indMessageAsn1Bytes)
		if err != nil {
			log.Errorf("Error - converting asn1 bytes to proto bytes: %s", err)
		}

		indMessage := e2sm_kpm_ies.E2SmKpmIndicationMessage{}
		err = proto.Unmarshal(indMessageProtoBytes, &indMessage)
		if err != nil {
			log.Errorf("Error - unmashal protobytes to struct: %s", err)
		}
		log.Infof("numUEs: %v", indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetCuCpResourceStatus().GetNumberOfActiveUes())
		log.Infof("CUCP Name: %v", indMessage.GetIndicationMessageFormat1().GetPmContainers()[0].GetPerformanceContainer().GetOCuCp().GetGNbCuCpName().GetValue())
	}
}
