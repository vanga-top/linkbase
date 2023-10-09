package runtime

import (
	"context"
	"google.golang.org/grpc/metadata"
)

// ServerMetadata consists of metadata sent from gRPC server.
type ServerMetadata struct {
	HeaderMD  metadata.MD
	TrailerMD metadata.MD
}

type serverMetadataKey struct{}

// ServerMetadataFromContext returns the ServerMetadata in ctx
func ServerMetadataFromContext(ctx context.Context) (md ServerMetadata, ok bool) {
	if ctx == nil {
		return md, false
	}
	md, ok = ctx.Value(serverMetadataKey{}).(ServerMetadata)
	return
}