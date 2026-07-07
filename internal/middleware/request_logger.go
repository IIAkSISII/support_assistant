package middleware

import (
	"github.com/IIAkSISII/support-assistant/observability"
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		return
	}

	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	written, err := r.ResponseWriter.Write(data)
	r.bytesWritten += int64(written)

	return written, err
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			resourcesBefore := observability.CaptureResources()

			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)

			statusCode := recorder.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			resourceUsage := resourcesBefore.Diff(observability.CaptureResources())

			logger.Info(
				"http request processed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", statusCode,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"content_length", r.ContentLength,
				"response_size_bytes", recorder.bytesWritten,
				"resource_heap_alloc_bytes", resourceUsage.HeapAllocBytes,
				"resource_heap_alloc_delta_bytes", resourceUsage.HeapAllocDeltaBytes,
				"resource_heap_sys_bytes", resourceUsage.HeapSysBytes,
				"resource_total_alloc_delta_bytes", resourceUsage.TotalAllocDeltaBytes,
				"resource_gc_delta", resourceUsage.GCDelta,
				"resource_goroutines", resourceUsage.Goroutines,
				"resource_goroutines_delta", resourceUsage.GoroutinesDelta,
			)
		})
	}
}
