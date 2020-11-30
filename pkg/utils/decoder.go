package utils

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/kpmctypes"
	e2sm_kpm_ies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
)

func IndicationHeaderASN1toProto(asn1Bytes []byte) ([]byte, error) {
	perBytes, err := kpmctypes.PerDecodeE2SmKpmIndicationHeader(asn1Bytes)
	if err != nil {
		return nil, fmt.Errorf("error decoding E2SmKpmIndicationHeader to PER %s", err)
	}

	protoBytes, err := proto.Marshal(perBytes)
	if err != nil {
		return nil, fmt.Errorf("error marshalling asn1Bytes to E2SmKpmIndicationHeader %s", err)
	}

	return protoBytes, nil
}

func IndicationHeaderProtoToASN1(protoBytes []byte) ([]byte, error) {
	protoObj := new(e2sm_kpm_ies.E2SmKpmIndicationHeader)
	if err := proto.Unmarshal(protoBytes, protoObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling protoBytes to E2SmKpmIndicationHeader %s", err)
	}

	perBytes, err := kpmctypes.PerEncodeE2SmKpmIndicationHeader(protoObj)
	if err != nil {
		return nil, fmt.Errorf("error encoding E2SmKpmIndicationHeader to PER %s", err)
	}

	return perBytes, nil
}

func IndicationMessageASN1toProto(asn1Bytes []byte) ([]byte, error) {
	perBytes, err := kpmctypes.PerDecodeE2SmKpmIndicationMessage(asn1Bytes)
	if err != nil {
		return nil, fmt.Errorf("error decoding E2SmKpmIndicationMessage to PER %s", err)
	}

	protoBytes, err := proto.Marshal(perBytes)
	if err != nil {
		return nil, fmt.Errorf("error marshalling asn1Bytes to E2SmKpmIndicationMessage %s", err)
	}

	return protoBytes, nil
}

func IndicationMessageProtoToASN1(protoBytes []byte) ([]byte, error) {
	protoObj := new(e2sm_kpm_ies.E2SmKpmIndicationMessage)
	if err := proto.Unmarshal(protoBytes, protoObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling protoBytes to E2SmKpmIndicationMessage %s", err)
	}

	perBytes, err := kpmctypes.PerEncodeE2SmKpmIndicationMessage(protoObj)
	if err != nil {
		return nil, fmt.Errorf("error encoding E2SmKpmIndicationMessage to PER %s", err)
	}

	return perBytes, nil
}