package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInit(t *testing.T) {
	mgr, err := Init()
	assert.EqualError(t, err, "init error")

	_, err = mgr.GetConfig("test.env")
	assert.EqualError(t, err, "get config error")
}
