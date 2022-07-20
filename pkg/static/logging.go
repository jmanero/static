package static

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

// String Constants
const (
	// Based upon nginx log format: $remote_addr - $remote_user [$time_local] "$request" $status $request_time $body_bytes_sent "$http_referer" "$http_user_agent"
	AccessLogFormat = `%s - %s [%s] "%s %s %s" %d %s %d "%s" "%s"`
)

// Loggers
var (
	Access = log.New(os.Stdout, "", 0)
	Errors = log.New(os.Stderr, "", log.LUTC|log.Ldate|log.Ltime)
)

// LogAccess is a helper to log requests
func LogAccess(w *WriteCounter, r *http.Request, code int) {
	username := r.URL.User.Username()
	if len(username) == 0 {
		username = "-"
	}

	Access.Printf(AccessLogFormat,
		r.RemoteAddr,                    // $remote_addr
		username,                        // $remote_user
		time.Now().UTC(),                // $time_local
		r.Method, r.RequestURI, r.Proto, // $request
		code,          // $status
		w.Duration(),  // $request_time
		w.Size,        // $body_bytes_sent
		r.Referer(),   // $http_referer
		r.UserAgent(), // $http_user_agent
	)
}

// ServeError is a helper to write and log error responses
func ServeError(w *WriteCounter, r *http.Request, err error) {
	if errors.Is(err, os.ErrNotExist) {
		// Map ErrNotExist to a 404 response
		http.NotFound(w, r)
		LogAccess(w, r, http.StatusNotFound)
		return
	}

	if errors.Is(err, os.ErrPermission) {
		// Map ErrPermission to a 404 response and log the underlying error
		http.NotFound(w, r)
		LogAccess(w, r, http.StatusNotFound)
		Errors.Println(r.RemoteAddr, r.Method, r.RequestURI, r.Proto, err)
		return
	}

	// Respond with 500 and log error details internally
	http.Error(w, "Internal Server Error", 500)
	LogAccess(w, r, http.StatusInternalServerError)
	Errors.Println(r.RemoteAddr, r.Method, r.RequestURI, r.Proto, err)
}
