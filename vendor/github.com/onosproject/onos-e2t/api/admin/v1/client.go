// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package admin

import (
	"google.golang.org/grpc"
)

// E2TAdminClientFactory : Default E2AdminClient creation.
var E2TAdminClientFactory = func(cc *grpc.ClientConn) E2TAdminServiceClient {
	return NewE2TAdminServiceClient(cc)
}

// CreateE2AdminServiceClient creates and returns a new config admin client
func CreateE2TAdminServiceClient(cc *grpc.ClientConn) E2TAdminServiceClient {
	return E2TAdminClientFactory(cc)
}
