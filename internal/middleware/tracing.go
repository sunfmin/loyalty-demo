package middleware

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Tracing middleware extracts or starts OpenTracing spans for all HTTP requests
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract span context from incoming request headers
		spanCtx, _ := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)

		// Start a new span (or continue existing trace)
		span := opentracing.StartSpan(
			r.Method+" "+r.URL.Path,
			ext.RPCServerOption(spanCtx),
		)
		defer span.Finish()

		// Set standard HTTP span tags
		ext.HTTPMethod.Set(span, r.Method)
		ext.HTTPUrl.Set(span, r.URL.String())
		ext.Component.Set(span, "http")

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Add span to request context
		ctx := opentracing.ContextWithSpan(r.Context(), span)

		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Set HTTP status code tag after response
		ext.HTTPStatusCode.Set(span, uint16(wrapped.statusCode))
		if wrapped.statusCode >= 400 {
			ext.Error.Set(span, true)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

