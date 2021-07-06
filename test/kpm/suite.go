// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package kpm

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-kpimon/test/utils"
)

type TestSuite struct {
	sdran *helm.HelmRelease
	test.Suite
}

func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	// write files
	err := utils.WriteFile("/tmp/tls.cacrt", utils.TlsCacrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.crt", utils.TlsCrt)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/tls.key", utils.TlsKey)
	if err != nil {
		return err
	}
	err = utils.WriteFile("/tmp/config.json", utils.ConfigJson)
	if err != nil {
		return err
	}

	sdran, err := utils.CreateSdranRelease(c)
	if err != nil {
		return err
	}
	s.sdran = sdran
	sdran.Set("ran-simulator.pci.metricName", "metric").
		Set("ran-simulator.pci.modelName", "model")
	return sdran.Install(true)
}

// TearDownTestSuite uninstalls helm chart released
func (s *TestSuite) TearDownTestSuite() error {
	return s.sdran.Uninstall()
}
