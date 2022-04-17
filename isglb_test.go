package isglb

import (
	"github.com/yindaheng98/isglb/algorithms"
	"testing"
	"time"
)
import "github.com/yindaheng98/isglb/algorithms/impl/random"

func startISGLBServer() error {
	isglb := New(func() algorithms.Algorithm { return &random.Random{} })
	return isglb.Start(Config{
		Global: global{Dc: "dc1"},
		Log:    logConf{Level: "DEBUG"},
		Nats:   natsConf{URL: "nats://192.168.1.2:4222"},
	})
}

func TestISGLB(t *testing.T) {
	err := startISGLBServer()
	if err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Second)
}
