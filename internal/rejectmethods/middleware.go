package rejectmethods

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var acceptedMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// NewMiddleware returns middleware which rejects all unknown http methods
func NewMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if acceptedMethods[r.Method] {
			handler.ServeHTTP(w, r)
		} else {
			metrics.RejectedRequestsCount.Inc()
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	})
}
