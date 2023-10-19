package gateway

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRun(t *testing.T) {
	err := Run(context.Background(), Options{
		Addr:       "127,0,0,1",
		GRPCServer: Endpoint{},
		OpenAPIDir: "",
		Mux:        nil,
	})

	assert.NoError(t, err)
}
