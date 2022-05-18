package sxu

import (
	"fmt"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/spf13/viper"
	"github.com/yindaheng98/dion/config"
	"os"
)

// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

const (
	portRangeLimit = 100
)

// Config for sfu node
type Config struct {
	config.Common
	isfu.Config
}

func unmarshal(rawVal interface{}) error {
	if err := viper.Unmarshal(rawVal); err != nil {
		return err
	}
	return nil
}

func (c *Config) Load(file string) error {
	_, err := os.Stat(file)
	if err != nil {
		return err
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("config file %s read failed. %v\n", file, err)
		return err
	}

	err = unmarshal(c)
	if err != nil {
		return err
	}
	err = unmarshal(&c.Config)
	if err != nil {
		return err
	}
	if err != nil {
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

// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
