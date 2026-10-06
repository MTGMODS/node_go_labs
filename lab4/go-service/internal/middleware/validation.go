package middleware

import (
	"mime"
	"net/http"

	"node-go-labs/lab4/go-service/internal/httpjson"
)

const maxBodyBytes = 32 * 1024

func ValidateJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				httpjson.WriteError(w, http.StatusBadRequest, "invalid_content_type", "Content-Type must be application/json")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}
