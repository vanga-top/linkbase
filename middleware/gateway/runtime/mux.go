package runtime

import "net/http"

type ServeMux struct {
}

type ServeMuxOption func(mux *ServeMux)

type HandlerFunc func(w http.ResponseWriter, r *http.Request, pathParams map[string]string)

type handler struct {
	pat Pattern
	h   HandlerFunc
}
