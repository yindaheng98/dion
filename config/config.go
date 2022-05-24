package config

import (
	log "github.com/pion/ion-log"
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
	Global Global   `mapstructure:"global"`
	Log    LogConf  `mapstructure:"log"`
	Nats   NatsConf `mapstructure:"nats"`
}

func LoadFromFile(file, ftype string, rawVal interface{}) error {
	_, err := os.Stat(file)
	if err != nil {
		return err
	}

	viper.SetConfigFile(file)
	viper.SetConfigType(ftype)

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

func LoadFromToml(file string, rawVal interface{}) error {
	return LoadFromFile(file, "toml", rawVal)
}

func (c *Common) Load(file string) error {
	return LoadFromToml(file, c)
}
