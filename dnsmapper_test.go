package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainParse(t *testing.T) {
	*flagdomain = "mapper.example.com"
	setup()

	assert.Equal(t, "", getUuidFromDomain("abc"), "invalid domain")
	assert.Equal(t, "", getUuidFromDomain("mapper.example.com"), "base domain")
	assert.Equal(t, "foobar", getUuidFromDomain("foobar.mapper.example.com"), "base domain with host")
}
