package runtime

import (
	"context"
	"github.com/cockroachdb/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
)

type ErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, error)

type StreamErrorHandlerFunc func(context.Context, error) *status.Status

type RoutingErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, error)

type HTTPStatusError struct {
	HTTPStatus int
	Err        error
}

func (e *HTTPStatusError) Error() string {
	return e.Err.Error()
}

func DefaultHttpErrorHandler(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	const fallback = `{"code": 13, "message": "failed to marshal error message"}`
	var customStatus *HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}
	s := status.Convert(err)
	pb := s.Proto()

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")
	contentType := marshaler.ContentType(pb)
	w.Header().Set("Content-Type", contentType)

	if s.Code() == codes.Unauthenticated {
		w.Header().Set("WWW-Authenticate", s.Message())
	}

	buf, merr := marshaler.Marshal(pb)
	if merr != nil {
		grpclog.Infof("Failed to marshal error message %q: %v", s, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
		}
		return
	}

	md, ok := ServerMetadataFromContext(ctx)
	if !ok {
		grpclog.Infof("Failed to extract ServerMetadata from context")
	}

}
