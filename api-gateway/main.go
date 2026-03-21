package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	monolithURL, _ := url.Parse("http://localhost:8080")
	userServiceURL, _ := url.Parse("http://localhost:8081")
	orderServiceURL, _ := url.Parse("http://localhost:8082")

	monolithProxy := httputil.NewSingleHostReverseProxy(monolithURL)
	userServiceProxy := httputil.NewSingleHostReverseProxy(userServiceURL)
	orderServiceProxy := httputil.NewSingleHostReverseProxy(orderServiceURL)

	r := gin.Default()

	// 1. Strangler Routing Logic
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		// Rule 1: All Writes (POST, PUT, DELETE) go to the Monolith
		if method != http.MethodGet {
			log.Printf("[PROXY] Write operation: %s %s -> Monolith", method, path)
			monolithProxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Rule 2: GET /users/:id -> User Service with Fallback
		if strings.HasPrefix(path, "/users/") {
			log.Printf("[PROXY] GET User: %s -> Attempting User Service", path)
			proxyWithFallback(c, userServiceProxy, monolithProxy)
			return
		}

		// Rule 3: GET /orders/:id -> Order Service with Fallback
		if strings.HasPrefix(path, "/orders/") {
			log.Printf("[PROXY] GET Order: %s -> Attempting Order Service", path)
			proxyWithFallback(c, orderServiceProxy, monolithProxy)
			return
		}

		// Rule 4: Everything else -> Monolith
		log.Printf("[PROXY] Default: %s %s -> Monolith", method, path)
		monolithProxy.ServeHTTP(c.Writer, c.Request)
	})

	log.Println("API Gateway (Strangler) starting on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Could not start API Gateway: %v", err)
	}
}

// proxyWithFallback attempts to use the primary proxy, but falls back to monolith if 404
func proxyWithFallback(c *gin.Context, primary *httputil.ReverseProxy, fallback *httputil.ReverseProxy) {
	writer := &statusSyncWriter{ResponseWriter: c.Writer, status: http.StatusOK}
	primary.ServeHTTP(writer, c.Request)

	if writer.status == http.StatusNotFound {
		log.Printf("[FALLBACK] 404 from Primary Service. Redirecting %s to Monolith", c.Request.URL.Path)
		// CRITICAL: We must reset any headers set by the first proxy attempt (like Content-Length)
		// before invoking the second proxy, otherwise curl gets malformed headers.
		for k := range c.Writer.Header() {
			c.Writer.Header().Del(k)
		}
		fallback.ServeHTTP(c.Writer, c.Request)
	}
}

type statusSyncWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusSyncWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *statusSyncWriter) WriteHeader(code int) {
	w.status = code
	if code != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusSyncWriter) Write(b []byte) (int, error) {
	if w.status == http.StatusNotFound {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
