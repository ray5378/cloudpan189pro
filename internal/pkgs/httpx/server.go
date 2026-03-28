package httpx

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}

func parseDurationEnv(key, def string) time.Duration {
	v := getenvDefault(key, def)
	d, err := time.ParseDuration(v)
	if err != nil { return mustDuration(def) }
	return d
}

func mustDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

func parseIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil { return n }
	}
	return def
}

// NewServer creates *http.Server with long timeouts friendly to WebDAV large transfers
// Defaults (overridable by env):
//   HTTP_READ_HEADER_TIMEOUT = 30s
//   HTTP_READ_TIMEOUT        = 5h
//   HTTP_WRITE_TIMEOUT       = 5h
//   HTTP_IDLE_TIMEOUT        = 10m
//   HTTP_MAX_HEADER_BYTES    = 1048576
func NewServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: parseDurationEnv("HTTP_READ_HEADER_TIMEOUT", "30s"),
		ReadTimeout:       parseDurationEnv("HTTP_READ_TIMEOUT", "5h"),
		WriteTimeout:      parseDurationEnv("HTTP_WRITE_TIMEOUT", "5h"),
		IdleTimeout:       parseDurationEnv("HTTP_IDLE_TIMEOUT", "10m"),
		MaxHeaderBytes:    parseIntEnv("HTTP_MAX_HEADER_BYTES", 1<<20),
	}
}
