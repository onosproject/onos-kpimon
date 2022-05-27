// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"
	"time"

	"github.com/onosproject/onos-kpimon/pkg/manager"
	"github.com/onosproject/onos-kpimon/test/utils"
	"github.com/onosproject/onos-lib-go/pkg/certs"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

var (
	waitPeriod                = time.Duration(30)
	reportPeriodIntervalValue = "5000"
	reportPeriodInterval      = "/report_period/interval"
	targetName                = "kpimon"
)

// TestKpmConfig is the function for Helmit-based integration test
func (s *TestSuite) TestKpmConfig(t *testing.T) {
	cfg := manager.Config{
		CAPath:      "/tmp/tls.cacrt",
		KeyPath:     "/tmp/tls.key",
		CertPath:    "/tmp/tls.crt",
		ConfigPath:  "/tmp/config.json",
		E2tEndpoint: "onos-e2t:5150",
		GRPCPort:    5150,
		RicActionID: 10,
		SMName:      utils.KpmServiceModelName,
		SMVersion:   utils.KpmServiceModelVersion,
	}

	_, err := certs.HandleCertPaths(cfg.CAPath, cfg.KeyPath, cfg.CertPath, true)
	assert.NoError(t, err)

	mgr := manager.NewManager(cfg)
	mgr.Run()

	ctx, cancel := context.WithTimeout(context.Background(), utils.TestTimeout)
	defer cancel()

	err = utils.WaitForKPMIndicationMessages(ctx, t, mgr)
	assert.NoError(t, err)

	ctx, cancel = gnmiutils.MakeContext()
	defer cancel()

	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)
	targetPath := gnmiutils.GetTargetPathWithValue(targetName, reportPeriodInterval, reportPeriodIntervalValue, proto.IntVal)

	// Set a value using onos-config
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPath,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	setReq.SetOrFail(t)

	time.Sleep(waitPeriod * time.Second)

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Paths:      targetPath,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValues(t, reportPeriodIntervalValue)

	t.Log("KPM suite test passed")
}
