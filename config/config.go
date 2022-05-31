package config

import (
	"bytes"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	log "github.com/pion/ion-log"
	"github.com/spf13/viper"
	"os"
	"reflect"
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

func clearNil(raw map[string]interface{}) map[string]interface{} {
	for k, v := range raw {
		if v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil()) {
			delete(raw, k) // 居然还有nil函数处理不了，放弃治疗，还不如直接写个脚本改配置文件
		}
	}
	return raw
}

func MergeTomlFromEnv(pre string, rawVal interface{}) error {
	var m map[string]interface{}
	err := mapstructure.Decode(rawVal, &m)
	if err != nil {
		log.Errorf("mapstructure decode failed. %v\n", err)
		return err
	}
	// mapstructure.Decode怎么连不打标记的地方都解析的
	// 出来直接nil，搞得后面的toml没法解析
	m = clearNil(m) // 还得我帮它清除
	b, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	viper.SetConfigType("toml")
	if err := viper.MergeConfig(bytes.NewReader(b)); err != nil {
		return err
	}
	viper.AutomaticEnv()
	viper.SetEnvPrefix(pre)
	if err := viper.Unmarshal(rawVal); err != nil {
		log.Errorf("config env unmarshal failed. %v\n", err)
		return err
	}

	log.Infof("config from environment load ok!")
	return nil
}

func (c *Common) Load(file string) error {
	return LoadFromToml(file, c)
}
