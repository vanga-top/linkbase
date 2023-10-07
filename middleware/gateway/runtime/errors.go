package runtime

import (
	"context"
	"google.golang.org/grpc/status"
	"net/http"
)

type ErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, error)

type StreamErrorHandlerFunc func(context.Context, error) *status.Status

type RoutingErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, error)
