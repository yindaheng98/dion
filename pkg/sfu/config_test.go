package sfu

import (
	"fmt"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	c := Config{}
	err := c.Load("D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%+v", c)
}
