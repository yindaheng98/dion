package sfu

import (
	"fmt"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/spf13/viper"
	"github.com/yindaheng98/dion/config"
)

const (
	portRangeLimit = 100
)

// Config for sfu node
type Config struct {
	config.Common
	isfu.Config
}

func (c *Config) Load(file string) error {
	err := c.Common.Load(file)
	if err != nil {
		log.Errorf("config file %s loaded failed. %v\n", file, err)
		return err
	}
	if err := viper.Unmarshal(&c.Config); err != nil {
		log.Errorf("config file %s loaded failed. %v\n", file, err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min,max]", file)
		log.Errorf("err=%v", err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] < portRangeLimit {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min, max] and max - min >= %d", file, portRangeLimit)
		log.Errorf("err=%v", err)
		return err
	}

	log.Infof("config %s load ok!", file)
	return nil
}
