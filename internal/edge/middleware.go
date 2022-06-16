package edge

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		rw := NewLoggingResponseWriter(w)
		next.ServeHTTP(rw, r)
		elapsedTime := time.Since(startTime)

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		log.Println(
			fmt.Sprintf(
				"%s - \"%s %s %s\" %s %d %v",
				host,
				r.Method,
				r.RequestURI,
				r.Proto,
				r.UserAgent(),
				rw.statusCode,
				elapsedTime,
			),
		)
	})
}
