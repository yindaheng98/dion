package sxu

import (
	log "github.com/pion/ion-log"
	ion_sfu_log "github.com/pion/ion-sfu/pkg/logger"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	pbion "github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/dion/pkg/sxu/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	"github.com/yindaheng98/dion/util"
)

const ServiceSXU = "sxu"

var logrLogger = ion_sfu_log.New().WithName("dion-sxu-node")

func init() {
	ion_sfu_log.SetGlobalOptions(ion_sfu_log.GlobalConfig{V: 1})
	ion_sfu.Logger = logrLogger.WithName("sxu")
}

type Config struct {
	sfu.Config
	ISFU ion_sfu.Config `mapstructure:"isfu"`
}

type SXU struct {
	ion.Node
	syncer *syncer.ISGLBSyncer
	server *sfu.SFUServer
	isfu   *ion_sfu.SFU
	conf   Config
}

func NewSXU() *SXU {
	return &SXU{
		Node: ion.NewNode("sxu-" + util.RandomString(8)),
	}
}

// Load load config file
func (s *SXU) Load(confFile string) error {
	err := s.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

func (s *SXU) Start(conf Config) error {
	return s.StartWithBuilder(conf, DefaultToolBoxBuilder{})
}

func (s *SXU) StartWithBuilder(conf Config, toolbox ToolBoxBuilder) error {
	// Start internal SFU
	s.isfu = ion_sfu.NewSFU(conf.ISFU)

	// Start SFU Signal service
	s.server = sfu.NewSFUServer(&s.Node, s.isfu)
	err := s.server.Start(conf.Config)
	if err != nil {
		return err
	}

	// Start syncer
	s.syncer = syncer.NewSFUStatusSyncer(&s.Node, "*", &pbion.Node{
		Dc:      conf.Global.Dc,
		Nid:     s.Node.NID,
		Service: ServiceSXU,
		Rpc:     nil,
	}, toolbox.Build(&s.Node, s.isfu))
	s.syncer.Start()

	return nil
}

// Close all
func (s *SXU) Close() {
	s.syncer.Stop()
	s.server.Close()
}
