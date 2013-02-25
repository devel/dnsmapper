package main

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ConfigSuite struct {
}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) TestRedis(c *C) {
	redisConnect()
	Redis.Set("test", "hoopla")
}
