package config

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/runner"
	"github.com/spf13/viper"
	"os"
)

type Global struct {
	Dc string `mapstructure:"dc"`
}

type NatsConf struct {
	URL string `mapstructure:"url"`
}

type LogConf struct {
	Level string `mapstructure:"level"`
}

type Common struct {
	runner.ConfigBase
	Global Global   `mapstructure:"global"`
	Log    LogConf  `mapstructure:"log"`
	Nats   NatsConf `mapstructure:"nats"`
}

func Load(file string, rawVal interface{}) error {
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

	if err := viper.Unmarshal(rawVal); err != nil {
		log.Errorf("config file %s unmarshal failed. %v\n", file, err)
		return err
	}

	log.Infof("config %s load ok!", file)
	return nil
}

func (c *Common) Load(file string) error {
	return Load(file, c)
}
