package runtime

import (
	"context"
	"google.golang.org/protobuf/proto"
	"net/http"
)

type ServeMux struct {
	handlers               map[string][]handler
	forwardResponseOptions []func(context.Context, http.ResponseWriter, proto.Message)
}

type ServeMuxOption func(mux *ServeMux)

type HandlerFunc func(w http.ResponseWriter, r *http.Request, pathParams map[string]string)

type handler struct {
	pat Pattern
	h   HandlerFunc
}
