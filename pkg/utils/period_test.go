// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package utils

import (
	e2smkpmies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

var periodRanges = PeriodRanges{
	{0, 10, 0},
	{11, 20, 1},
	{21, 32, 2},
	{33, 40, 3},
	{41, 60, 4},
	{61, 64, 5},
	{65, 70, 6},
	{71, 80, 7},
	{81, 128, 8},
	{129, 160, 9},
	{161, 256, 10},
	{257, 320, 11},
	{321, 512, 12},
	{513, 640, 13},
	{641, 1024, 14},
	{1025, 1280, 15},
	{1281, 2048, 16},
	{2049, 2560, 17},
	{2561, 5120, 18},
	{5121, math.MaxInt64, 19},
}

func TestPeriodRanges_Search(t *testing.T) {
	id1 := periodRanges.Search(0)
	assert.Equal(t, e2smkpmies.RtPeriodIe_RT_PERIOD_IE_MS10, id1)
	id2 := periodRanges.Search(30)
	assert.Equal(t, e2smkpmies.RtPeriodIe_RT_PERIOD_IE_MS32, id2)
	id3 := periodRanges.Search(100000)
	assert.Equal(t, e2smkpmies.RtPeriodIe_RT_PERIOD_IE_MS10240, id3)
}
