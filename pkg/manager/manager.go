// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	appConfig "github.com/onosproject/onos-kpimon/pkg/config"
	"github.com/onosproject/onos-kpimon/pkg/monitoring"
	nbi "github.com/onosproject/onos-kpimon/pkg/northbound"
	"github.com/onosproject/onos-kpimon/pkg/rnib"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/e2"
	"github.com/onosproject/onos-kpimon/pkg/southbound/e2/subscription"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
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
	RicActionID   int32
	SMName        string
	SMVersion     string
}

// NewManager generates the new KPIMON xAPP manager
func NewManager(config Config) *Manager {
	indCh := make(chan indication.Indication)
	metricStore := make(map[int]string)
	appCfg, err := appConfig.NewConfig()
	if err != nil {
		log.Warn(err)
	}

	e2Client := e2.NewE2Client(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID, config.SMName, config.SMVersion, metricStore, appCfg)

	subManager, err := subscription.NewManager(
		subscription.WithE2SubAddress("onos-e2sub", 5150),
		subscription.WithE2TAddress("onos-e2t", 5150),
		subscription.WithServiceModel("oran-e2sm-kpm", "v2"))

	if err != nil {
		log.Warn(err)
	}

	manager := &Manager{
		config:       config,
		indChan:      indCh,
		adminSession: admin.NewE2AdminSession(config.E2tEndpoint),
		e2Client:     e2Client,
		monitor:      monitoring.NewMonitor(indCh, appCfg),
		metricStore:  metricStore,
		subManager:   subManager,
	}
	return manager
}

// Manager is an abstract struct for manager
type Manager struct {
	appConfig    appConfig.Config
	config       Config
	e2Client     e2.E2Client
	adminSession admin.E2AdminSession
	topoClient   *rnib.Client
	monitor      *monitoring.Monitor
	metricStore  map[int]string
	indChan      chan indication.Indication
	subManager   subscription.Manager
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
	/*err := m.startNorthboundServer()
	if err != nil {
		return err
	}*/

	err := m.subManager.Start()
	if err != nil {
		return err
	}

	//go m.e2Client.Run(m.indChan, m.adminSession)
	//go m.monitor.Run(m.metricStore)

	return nil
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.config.CAPath,
		m.config.KeyPath,
		m.config.CertPath,
		int16(m.config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	s.AddService(nbi.NewService(m.monitor))

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
