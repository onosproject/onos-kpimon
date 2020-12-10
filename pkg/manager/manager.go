// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package manager

import (
	"github.com/onosproject/onos-kpimon/pkg/controller"
	"github.com/onosproject/onos-kpimon/pkg/southbound/admin"
	"github.com/onosproject/onos-kpimon/pkg/southbound/ricapie2"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-ric-sdk-go/pkg/e2/indication"
	"github.com/onosproject/onos-ric-sdk-go/pkg/gnmi"
)

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath         string
	KeyPath        string
	CertPath       string
	E2tEndpoint    string
	E2SubEndpoint  string
	GRPCPort       int
	GnmiConfig     *gnmi.Config
	RicActionID    int32
	RicRequestorID int32
	RicInstanceID  int32
	RanFuncID      uint8
}

// NewManager creates a new manager
func NewManager(config Config) *Manager {
	log.Info("Creating Manager")
	indCh := make(chan indication.Indication)
	return &Manager{
		Config: config,
		Sessions: SBSessions{
			AdminSession: admin.NewSession(config.E2tEndpoint),
			E2Session:    ricapie2.NewSession(config.E2tEndpoint, config.E2SubEndpoint, config.RicActionID, config.RicRequestorID, config.RicInstanceID, config.RanFuncID),
		},
		Chans: Channels{
			IndCh: indCh, // Connection between KPIMON core and Southbound
		},
		Ctrls: Controllers{
			KpiMonCtrl: controller.NewKpiMonController(indCh),
		},
	}
}

// Manager is a manager for the KPIMON service
type Manager struct {
	Config   Config
	Sessions SBSessions
	Chans    Channels
	Ctrls    Controllers
}

// SBSessions is a set of Southbound sessions
type SBSessions struct {
	E2Session    *ricapie2.E2Session
	AdminSession *admin.E2AdminSession
}

// Channels is a set of channels
type Channels struct {
	IndCh chan indication.Indication
}

// Controllers is a set of controllers
type Controllers struct {
	KpiMonCtrl *controller.KpiMonCtrl
}

// Run starts the manager and the associated services
func (m *Manager) Run() {
	log.Info("Running Manager")
	if err := m.Start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

// Start starts the manager
func (m *Manager) Start() error {

	// Start Northbound server
	err := m.startNorthboundServer()
	if err != nil {
		return err
	}

	// Start Southbound client to watch indication messages
	go m.Sessions.E2Session.Run(m.Chans.IndCh, m.Sessions.AdminSession)
	go m.Ctrls.KpiMonCtrl.Run()
	return nil
}

// Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
}

func (m *Manager) startNorthboundServer() error {
	s := northbound.NewServer(northbound.NewServerCfg(
		m.Config.CAPath,
		m.Config.KeyPath,
		m.Config.CertPath,
		int16(m.Config.GRPCPort),
		true,
		northbound.SecurityConfig{}))

	gnmiAgent := gnmi.RegisterConfigurable(m.Config.GnmiConfig)

	// TODO add services including gnmi service
	s.AddService(gnmiAgent)

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
