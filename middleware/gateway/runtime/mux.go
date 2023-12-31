package runtime

import (
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/textproto"
	"strings"
)

// UnescapingMode defines the behavior of ServeMux when unescaping path parameters.
type UnescapingMode int

const (
	// UnescapingModeLegacy is the default V2 behavior, which escapes the entire
	// path string before doing any routing.
	UnescapingModeLegacy UnescapingMode = iota

	// UnescapingModeAllExceptReserved unescapes all path parameters except RFC 6570
	// reserved characters.
	UnescapingModeAllExceptReserved

	// UnescapingModeAllExceptSlash unescapes URL path parameters except path
	// separators, which will be left as "%2F".
	UnescapingModeAllExceptSlash

	// UnescapingModeAllCharacters unescapes all URL path parameters.
	UnescapingModeAllCharacters

	// UnescapingModeDefault is the default escaping type.
	// TODO(v3): default this to UnescapingModeAllExceptReserved per grpc-httpjson-transcoding's
	// reference implementation
	UnescapingModeDefault = UnescapingModeLegacy
)

type ServeMux struct {
	handlers                  map[string][]handler
	forwardResponseOptions    []func(context.Context, http.ResponseWriter, proto.Message) error
	marshalers                marshalerRegistry
	incomingHeaderMatcher     HeaderMatcherFunc
	outgoingHeaderMatcher     HeaderMatcherFunc
	metadataAnnotators        []func(context.Context, *http.Request) metadata.MD
	errorHandler              ErrorHandlerFunc
	streamErrorHandler        StreamErrorHandlerFunc
	routingErrorHandler       RoutingErrorHandlerFunc
	disablePathLengthFallback bool
	unescapingMode            UnescapingMode
}

func NewServerMux(opts ...ServeMuxOption) *ServeMux {
	serveMux := &ServeMux{
		handlers:               make(map[string][]handler),
		forwardResponseOptions: make([]func(context.Context, http.ResponseWriter, proto.Message) error, 0),
		marshalers:             makeMarshalerMIMERegistry(),
		errorHandler:           DefaultHttpErrorHandler,
		streamErrorHandler:     DefaultStreamErrorHandler,
		routingErrorHandler:    DefaultRoutingErrorHandler,
		unescapingMode:         UnescapingModeDefault,
	}

	for _, opt := range opts {
		opt(serveMux)
	}

	if serveMux.incomingHeaderMatcher == nil {
		serveMux.incomingHeaderMatcher = DefaultHeaderMatcher
	}

	if serveMux.outgoingHeaderMatcher == nil {
		serveMux.outgoingHeaderMatcher = func(key string) (string, bool) {
			return fmt.Sprintf("%s%s", MetadataHeaderPrefix, key), true
		}
	}

	return serveMux
}

func DefaultHeaderMatcher(key string) (string, bool) {
	switch key = textproto.CanonicalMIMEHeaderKey(key); {
	case isPermanentHTTPHeader(key):
		return MetadataPrefix + key, true
	case strings.HasPrefix(key, MetadataHeaderPrefix):
		return key[len(MetadataHeaderPrefix):], true
	}
	return "", false
}

type ServeMuxOption func(mux *ServeMux)

type HandlerFunc func(w http.ResponseWriter, r *http.Request, pathParams map[string]string)

type handler struct {
	pat Pattern
	h   HandlerFunc
}

type HeaderMatcherFunc func(string) (string, bool)
