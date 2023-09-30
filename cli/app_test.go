package cli

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApp_Run(t *testing.T) {
	err := NewApp().Run([]string{"server", "-h"})
	assert.Equal(t, nil, err)
}
