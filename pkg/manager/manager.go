// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	"github.com/onosproject/onos-kpimon/pkg/controller"
	nbi "github.com/onosproject/onos-kpimon/pkg/northbound"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-kpimon/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
	configurable "github.com/onosproject/onos-ric-sdk-go/pkg/config/registry"
	configutils "github.com/onosproject/onos-ric-sdk-go/pkg/config/utils"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath        string
	KeyPath       string
	CertPath      string
	E2tEndpoint   string
	E2SubEndpoint string
	GRPCPort      int
	AppConfig     *app.Config
	RicActionID   int32
	SMName        string
	SMVersion     string
}

// NewManager generates the new KPIMON xAPP manager
func NewManager(config Config) Manager {
	var manager Manager
	if config.SMVersion == "v1" {
		manager = newV1Manager(config)
	} else if config.SMVersion == "v2" {
		manager = newV2Manager(config)
	} else {
		log.Fatal("The received service model version %s is not valid - it must be v1 or v2", config.SMVersion)
	}
	return manager
}

// Manager is an interface of KPIMON manager
type Manager interface {
	Run()
	Close()
	start() error
	registerConfigurable() error
	startNorthboundServer() error
	getReportPeriod() (uint64, error)
}

// AbstractManager is an abstract struct for manager
type AbstractManager struct {
	Manager
	Config   Config
	Sessions SBSessions
	Chans    Channels
	Ctrls    Controllers
	Maps     Maps
}

// SBSessions is a set of Southbound sessions
type SBSessions struct {
	E2Session    ricapie2.E2Session
	AdminSession admin.E2AdminSession
}

// Channels is a set of channels
type Channels struct {
	IndCh chan indication.Indication
}

// Controllers is a set of controllers
type Controllers struct {
	KpiMonController controller.KpiMonController
}

// Maps is a set of Map
type Maps struct {
	KpiMonMetricMap map[int]string
}

// Run runs KPIMON manager
func (m *AbstractManager) Run() {
	err := m.start()
	if err != nil {
		log.Errorf("Error when starting KPIMON: %v", err)
	}
}

// Close closes manager
func (m *AbstractManager) Close() {
	log.Info("closing Manager")
}

func (m *AbstractManager) start() error {
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}

	err = m.registerConfigurable()
	if err != nil {
		log.Error("Failed to register the app as a configurable entity", err)
		return err
	}

	period, err := m.getReportPeriod()
	if err != nil {
		log.Errorf("Failed to get report period so period is set to 30ms: %v", err)
		period = 30
	}
	m.Sessions.E2Session.SetReportPeriodMs(period)
	m.Sessions.E2Session.SetAppConfig(m.Config.AppConfig)

	go m.Sessions.E2Session.Run(m.Chans.IndCh, m.Sessions.AdminSession)
	go m.Ctrls.KpiMonController.Run(m.Maps.KpiMonMetricMap)

	return nil
}

func (m *AbstractManager) registerConfigurable() error {
	appConfig, err := configurable.RegisterConfigurable(&configurable.RegisterRequest{})
	if err != nil {
		return err
	}
	m.Config.AppConfig = appConfig.Config.(*app.Config)
	return nil
}

func (m *AbstractManager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.Config.CAPath,
		m.Config.KeyPath,
		m.Config.CertPath,
		int16(m.Config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(nbi.NewService(m.Ctrls.KpiMonController))

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

func (m *AbstractManager) getReportPeriod() (uint64, error) {
	interval, _ := m.Config.AppConfig.Get(utils.ReportPeriodConfigPath)
	val, err := configutils.ToUint64(interval.Value)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	log.Infof("Received period value: %v", val)
	return val, nil
}
