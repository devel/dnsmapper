package storeapi

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type MainSuite struct {
}

var _ = Suite(&MainSuite{})

func (s *MainSuite) TestApi(c *C) {
}
