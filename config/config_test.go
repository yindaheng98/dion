package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	conf := &Common{}
	err := os.Setenv("TEST_GLOBAL.DC", "123456")
	if err != nil {
		t.Error(err)
	}
	err = MergeTomlFromEnv("TEST", conf)
	if err != nil {
		t.Error(err)
	}
}
