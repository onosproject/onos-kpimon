// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package ha

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-kpimon/test/utils"
	"time"
)

// TestSuite has sdran release and test suite
type TestSuite struct {
	sdran *helm.HelmRelease
	test.Suite
	c *input.Context
}

// SetupTestSuite prepares test suite setup
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	s.c = c
	// write files
	err := utils.WriteFile("/tmp/tls.cacrt", utils.TLSCacrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.crt", utils.TLSCrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.key", utils.TLSKey)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/config.json", utils.ConfigJSON)
	if err != nil {
		return err
	}

	sdran, err := utils.CreateSdranRelease(c)
	if err != nil {
		return err
	}
	s.sdran = sdran
	sdran.Set("ran-simulator.pci.metricName", "metric").
		Set("ran-simulator.pci.modelName", "model").
		Set("import.onos-kpimon.enabled", true).
		Set("import.ran-simulator.enabled", false).
		WithTimeout(5 * time.Minute)

	return sdran.Install(true)
}
