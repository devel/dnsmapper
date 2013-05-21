package main

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type MainSuite struct {
}

var _ = Suite(&MainSuite{})

func (s *MainSuite) TestRedis(c *C) {
	*flagdomain = "mapper.example.com"
	setup()

	c.Check(getUuidFromDomain("abc"), Equals, "")
	c.Check(getUuidFromDomain("mapper.example.com"), Equals, "")
	c.Check(getUuidFromDomain("foobar.mapper.example.com"), Equals, "foobar")

}
