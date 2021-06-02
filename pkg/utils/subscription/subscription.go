// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package subscription

import (
	subapi "github.com/onosproject/onos-api/go/onos/e2sub/subscription"
	"github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm_v2/pdubuilder"
	"google.golang.org/protobuf/proto"
)

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
