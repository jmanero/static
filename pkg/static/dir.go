package static

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Dir implements http.FileSystem much like http.Dir, without all of the scary
// security concerns documented therein, but note the less still present, no
// doubt for pesky notions of reverse comparability...
type Dir string

// String Constants
const (
	PathSeparator = "/"

	ContentEncoding = "Content-Encoding"
	ContentLength   = "Content-Length"
	ContentType     = "Content-Type"
	LastModified    = "Last-Modified"

	DefaultContentType = "application/octet-stream"
)

// WriteCounter wraps an http.ResponseWriter with a body-size counter
type WriteCounter struct {
	http.ResponseWriter

	Started time.Time
	Size    int
}

// NewWriteCounter wraps an http.ResponseWriter in a WriteCounter proxy
func NewWriteCounter(w http.ResponseWriter) *WriteCounter {
	return &WriteCounter{w, time.Now(), 0}
}

func (w *WriteCounter) Write(block []byte) (s int, err error) {
	s, err = w.ResponseWriter.Write(block)

	w.Size += s
	return
}

// Duration calculates the time elapsed since the beginning of request handling
func (w *WriteCounter) Duration() time.Duration {
	return time.Since(w.Started)
}

func (dir Dir) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Normalize the request path to a local filesystem path
	parts := []string{string(dir)}
	parts = append(parts, strings.Split(r.URL.Path, PathSeparator)...)

	target := filepath.Join(parts...)
	target = filepath.Clean(target)

	dir.ServeFile(NewWriteCounter(w), r, target)
}

// ServeFile attempts to stream the contents of a file to an HTTP response
func (dir Dir) ServeFile(w *WriteCounter, r *http.Request, target string) {
	// This is a rudimentary check to ensure that Clean has not walked a request path
	// outside of the root path via .. compression
	if !strings.HasPrefix(target, string(dir)) {
		ServeError(w, r, os.ErrNotExist) // 404
		return
	}

	// Do not follow symlinks unless their target is also nested in the root path
	stat, err := os.Lstat(target)
	if err != nil {
		// Automatically maps access/exist errors to 404
		ServeError(w, r, err)
		return
	}

	// TODO: Check for request preconditions and ranges
	mode := stat.Mode()

	if mode.IsRegular() {
		// Fast path: Serve a regular file
		handle, err := os.Open(target)
		if err != nil {
			ServeError(w, r, err)
			return
		}

		defer handle.Close()

		if w.Header().Get(ContentEncoding) == "" {
			// Set Content-Length unless chunked encoding has been specified
			w.Header().Set(ContentLength, strconv.FormatInt(stat.Size(), 10))
		}

		if w.Header().Get(ContentType) == "" {
			// Attempt to detect content type from the target's extension
			detected := mime.TypeByExtension(filepath.Ext(stat.Name()))
			if len(detected) == 0 {
				detected = DefaultContentType
			}

			w.Header().Set(ContentType, detected)
		}

		// Always set Last-Modified from the underlying file
		w.Header().Set(LastModified, stat.ModTime().Format(time.RFC1123))

		// Write the response
		w.WriteHeader(http.StatusOK)
		io.Copy(w, handle)
		LogAccess(w, r, http.StatusOK)

		return
	}

	if mode&fs.ModeSymlink != 0 {
		// Try to follow a symlink that resolves to another file in the root path
		ltarget, err := filepath.EvalSymlinks(target)
		if err != nil {
			ServeError(w, r, err)
			return
		}

		// Try to serve the resolved path. filepath.EvalSymlinks should never
		// return a path to another symlink.
		dir.ServeFile(w, r, ltarget)
		return
	}

	// If target is not a regular file or symlink, refuse to serve it
	ServeError(w, r, os.ErrNotExist) // 404
}
