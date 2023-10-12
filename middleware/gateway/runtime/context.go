package runtime

import (
	"context"
	"google.golang.org/grpc/metadata"
)

// MetadataHeaderPrefix is the http prefix that represents custom metadata
// parameters to or from a gRPC call.
const MetadataHeaderPrefix = "Grpc-Metadata-"

// MetadataPrefix is prepended to permanent HTTP header keys (as specified
// by the IANA) when added to the gRPC context.
const MetadataPrefix = "grpcgateway-"

// MetadataTrailerPrefix is prepended to gRPC metadata as it is converted to
// HTTP headers in a response handled by grpc-gateway
const MetadataTrailerPrefix = "Grpc-Trailer-"

const metadataGrpcTimeout = "Grpc-Timeout"
const metadataHeaderBinarySuffix = "-Bin"

const xForwardedFor = "X-Forwarded-For"
const xForwardedHost = "X-Forwarded-Host"

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

func isPermanentHTTPHeader(hdr string) bool {
	switch hdr {
	case
		"Accept",
		"Accept-Charset",
		"Accept-Language",
		"Accept-Ranges",
		"Authorization",
		"Cache-Control",
		"Content-Type",
		"Cookie",
		"Date",
		"Expect",
		"From",
		"Host",
		"If-Match",
		"If-Modified-Since",
		"If-None-Match",
		"If-Schedule-Tag-Match",
		"If-Unmodified-Since",
		"Max-Forwards",
		"Origin",
		"Pragma",
		"Referer",
		"User-Agent",
		"Via",
		"Warning":
		return true
	}
	return false
}