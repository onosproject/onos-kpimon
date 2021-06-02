// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	appConfig "github.com/onosproject/onos-kpimon/pkg/config"
	"github.com/onosproject/onos-kpimon/pkg/monitoring"
	nbi "github.com/onosproject/onos-kpimon/pkg/northbound"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	app "github.com/onosproject/onos-ric-sdk-go/pkg/config/app/default"
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
func NewManager(config Config) *Manager {
	indCh := make(chan indication.Indication)
	kpiMonMetricMap := make(map[int]string)
	appCfg, err := appConfig.NewConfig()
	if err != nil {
		log.Warn(err)
	}

	e2Session := ricapie2.NewE2Session(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID,
		0, config.SMName, config.SMVersion, kpiMonMetricMap, appCfg)

	if err != nil {
		log.Warn(err)
	}
	manager := &Manager{
		Config:    config,
		appConfig: appCfg,
		Chans: Channels{
			IndCh: indCh,
		},
		AdminSession: admin.NewE2AdminSession(config.E2tEndpoint),
		E2Session:    e2Session,

		Monitor: monitoring.NewMonitor(indCh, appCfg),
		Maps: Maps{
			KpiMonMetricMap: kpiMonMetricMap,
		},
	}
	return manager
}

// Manager is an abstract struct for manager
type Manager struct {
	appConfig    appConfig.Config
	Config       Config
	E2Session    ricapie2.E2Session
	AdminSession admin.E2AdminSession
	Chans        Channels
	Monitor      *monitoring.Monitor
	Maps         Maps
}

// Channels is a set of channels
type Channels struct {
	IndCh chan indication.Indication
}

// Maps is a set of Map
type Maps struct {
	KpiMonMetricMap map[int]string
}

// Run runs KPIMON manager
func (m *Manager) Run() {
	err := m.start()
	if err != nil {
		log.Errorf("Error when starting KPIMON: %v", err)
	}
}

// Close closes manager
func (m *Manager) Close() {
	log.Info("closing Manager")
}

func (m *Manager) start() error {
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}

	go m.E2Session.Run(m.Chans.IndCh, m.AdminSession)
	go m.Monitor.Run(m.Maps.KpiMonMetricMap)

	return nil
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.Config.CAPath,
		m.Config.KeyPath,
		m.Config.CertPath,
		int16(m.Config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(nbi.NewService(m.Monitor))

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
