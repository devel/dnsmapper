package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainParse(t *testing.T) {
	*flagdomain = "mapper.example.com"
	setup()

	assert.Equal(t, "", getUUIDFromDomain("abc"), "invalid domain")
	assert.Equal(t, "", getUUIDFromDomain("mapper.example.com"), "base domain")
	assert.Equal(t, "foobar", getUUIDFromDomain("foobar.mapper.example.com"), "base domain with host")
}
