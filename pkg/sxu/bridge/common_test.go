package bridge

import (
	"github.com/yindaheng98/dion/pkg/sfu"
)

// readConf Read a Config
func readConf(confFile string) sfu.Config {
	conf := sfu.Config{}
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	return conf
}
