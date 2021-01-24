// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package utils

import (
	"sort"

	e2smkpmies "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_kpm/v1beta1/e2sm-kpm-ies"
)

const (
	// ReportPeriodConfigPath report period config path
	ReportPeriodConfigPath = "/report_period/interval"
)

// PeriodRange is a tuple of min and max value for each RT Period IE value
type PeriodRange struct {
	Min   int
	Max   int
	Value e2smkpmies.RtPeriodIe
}

// PeriodRanges is a set type of PeriodRange
type PeriodRanges []PeriodRange

// Len is the function to return period range length
func (r PeriodRanges) Len() int { return len(r) }

// Less is the function to check whether i is less than j
func (r PeriodRanges) Less(i, j int) bool { return r[i].Min < r[j].Min }

// Swap is the function to swap two elements
func (r PeriodRanges) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

// Sort is the function to sort period range
func (r PeriodRanges) Sort() { sort.Sort(r) }

// Search is the function to search a value in period range
func (r PeriodRanges) Search(v int) e2smkpmies.RtPeriodIe {
	rangesLength := r.Len()
	if i := sort.Search(rangesLength, func(i int) bool { return v <= r[i].Max }); i < rangesLength {
		if it := &r[i]; v >= it.Min && v <= it.Max {
			return it.Value
		}
	}
	return 0
}
