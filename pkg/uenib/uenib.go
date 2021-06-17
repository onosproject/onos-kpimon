// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package uenib

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-api/go/onos/uenib"
	"github.com/onosproject/onos-kpimon/pkg/store/event"
	measurementStore "github.com/onosproject/onos-kpimon/pkg/store/measurements"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/southbound"
)

const (
	// UENIBAddress has UENIB endpoint
	UENIBAddress = "onos-uenib:5150"
)

var log = logging.GetLogger("uenib")

// NewUENIBClient returns new UENIBClient object
func NewUENIBClient(ctx context.Context, certPath string, keyPath string, store measurementStore.Store) Client {
	conn, err := southbound.Connect(ctx, UENIBAddress, certPath, keyPath)

	if err != nil {
		log.Error(err)
	}
	return &client{
		client:           uenib.NewUEServiceClient(conn),
		measurementStore: store,
	}
}

// Client is an interface for UENIB client
type Client interface {
	// Run runs UENIBClient
	Run(ctx context.Context)

	// UpdateKPIMONResult updates KPIMON results to UENIB
	UpdateKPIMONResult(ctx context.Context, measEntry *measurementStore.Entry)

	// WatchMeasStore watches measurement entries
	WatchMeasStore(ctx context.Context, ch chan event.Event)
}

type client struct {
	client           uenib.UEServiceClient
	measurementStore measurementStore.Store
}

func (c *client) WatchMeasStore(ctx context.Context, ch chan event.Event) {
	for e := range ch {
		measEntry := e.Value.(*measurementStore.Entry)
		c.UpdateKPIMONResult(ctx, measEntry)
	}
}

func (c *client) Run(ctx context.Context) {
	ch := make(chan event.Event)
	err := c.measurementStore.Watch(ctx, ch)
	if err != nil {
		log.Warn(err)
	}

	go c.WatchMeasStore(ctx, ch)
}

func (c *client) UpdateKPIMONResult(ctx context.Context, measEntry *measurementStore.Entry) {
	req := c.createUENIBUpdateReq(measEntry)
	log.Debugf("UpdateReq msg: %v", req)
	resp, err := c.client.UpdateUE(ctx, req)
	if err != nil {
		log.Warn(err)
	}

	log.Debugf("resp: %v", resp)
}

func (c *client) createUENIBUpdateReq(measEntry *measurementStore.Entry) *uenib.UpdateUERequest {
	cellID := measEntry.Key.CellIdentity.CellID
	nodeID := measEntry.Key.NodeID
	keyID := fmt.Sprintf("%s:%s", nodeID, cellID)
	measEntryItems := measEntry.Value.([]measurementStore.MeasurementItem)
	uenibObj := uenib.UE{
		ID:      uenib.ID(keyID),
		Aspects: make(map[string]*types.Any),
	}
	log.Debugf("Key ID to be stored in UENIB: %v", keyID)
	log.Debugf("Meas Items to be stored in UENIB: %v", measEntryItems)

	for _, item := range measEntryItems {
		for _, record := range item.MeasurementRecords {
			switch val := record.MeasurementValue.(type) {
			case int64:
				strValue := fmt.Sprintf("%d", val)
				uenibObj.Aspects[record.MeasurementName] = &types.Any{
					TypeUrl: record.MeasurementName,
					Value:   []byte(strValue),
				}
			case float64:
				strValue := fmt.Sprintf("%f", val)
				uenibObj.Aspects[record.MeasurementName] = &types.Any{
					TypeUrl: record.MeasurementName,
					Value:   []byte(strValue),
				}
			case int32:
				strValue := fmt.Sprintf("%d", val)
				uenibObj.Aspects[record.MeasurementName] = &types.Any{
					TypeUrl: record.MeasurementName,
					Value:   []byte(strValue),
				}
			}
		}
	}
	return &uenib.UpdateUERequest{
		UE: uenibObj,
	}
}
