// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package measurements

// MeasurementItem measurement item
type MeasurementItem struct {
	MeasurementRecords []MeasurementRecord
}

// MeasurementRecord measurement record
type MeasurementRecord struct {
	Timestamp        uint64
	MeasurementName  string
	MeasurementValue interface{}
}

// CellIdentity is the ID for each cell
type CellIdentity struct {
	PlmnID string
	ECI    string
	CellID string
}
