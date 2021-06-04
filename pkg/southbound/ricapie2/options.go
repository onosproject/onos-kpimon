// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ricapie2

// Options E2 client options
type Options struct {
	E2TService E2TServiceOptions

	E2SubService E2SubServiceOptions
}

// E2TServiceOptions are the options for a E2T service
type E2TServiceOptions struct {
	// Host is the service host
	Host string
	// Port is the service port
	Port int
}

// E2SubServiceOptions are the options for E2sub service
type E2SubServiceOptions struct {
	// Host is the service host
	Host string
	// Port is the service port
	Port int
}
