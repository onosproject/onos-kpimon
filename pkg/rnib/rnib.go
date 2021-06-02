// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package rnib

import (
	"context"

	"github.com/onosproject/onos-ric-sdk-go/pkg/topo/options"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

// Client R-NIB client interface
type Client interface {
	WatchE2Nodes(ctx context.Context, ch chan topoapi.Event) error
	GetCells(ctx context.Context, nodeID topoapi.ID) ([]*topoapi.E2Cell, error)
}

// TopoClient topo SDK client
type TopoClient struct {
	client toposdk.Client
}

// GetCells get list of cells for each E2 node
func (c *TopoClient) GetCells(ctx context.Context, nodeID topoapi.ID) ([]*topoapi.E2Cell, error) {
	objects, err := c.client.List(ctx, options.WithListFilters(getContainsRelationFilter()))
	if err != nil {
		return nil, err
	}
	var cells []*topoapi.E2Cell

	for _, obj := range objects {
		relation := obj.Obj.(*topoapi.Object_Relation)
		if relation.Relation.SrcEntityID == nodeID {
			targetEntity := relation.Relation.TgtEntityID
			object, err := c.client.Get(ctx, targetEntity)
			if err != nil {
				return nil, err
			}
			if object != nil && object.GetEntity().GetKindID() == topoapi.ID(topoapi.RANEntityKinds_E2CELL.String()) {
				cellObject := &topoapi.E2Cell{}
				object.GetAspect(cellObject)
				cells = append(cells, cellObject)
			}
		}
	}

	return cells, nil

}

func getContainsRelationFilter() *topoapi.Filters {
	containsRelationFilter := &topoapi.Filters{
		KindFilters: []*topoapi.Filter{
			{
				Filter: &topoapi.Filter_Equal_{
					Equal_: &topoapi.EqualFilter{
						Value: topoapi.RANRelationKinds_CONTAINS.String(),
					},
				},
			},
		},
	}

	return containsRelationFilter

}

func getControlRelationFilter() *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		KindFilters: []*topoapi.Filter{
			{
				Filter: &topoapi.Filter_Equal_{
					Equal_: &topoapi.EqualFilter{
						Value: topoapi.RANRelationKinds_CONTROLS.String(),
					},
				},
			},
		},
	}
	return controlRelationFilter
}

// WatchE2Nodes watch e2 node changes
func (c *TopoClient) WatchE2Nodes(ctx context.Context, ch chan topoapi.Event) error {
	err := c.client.Watch(ctx, ch, options.WithWatchFilters(getControlRelationFilter()))
	if err != nil {
		return err
	}
	return nil
}

// NewClient creates a new topo SDK client
func NewClient() (*TopoClient, error) {
	sdkClient, err := toposdk.NewClient()
	if err != nil {
		return nil, err
	}
	cl := &TopoClient{
		client: sdkClient,
	}

	return cl, nil

}

var _ Client = &TopoClient{}
