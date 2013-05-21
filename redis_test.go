package main

import (
	. "launchpad.net/gocheck"
)

type ConfigSuite struct {
}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) TestRedis(c *C) {
	redisConnect()
	Redis.Set("test", "hoopla")
}
