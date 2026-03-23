package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	ServiceID string `json:"service_id"`
	ID        string `json:"id"`
	Result    string `json:"result"`
}

var (
	metricsLock sync.Mutex
	metrics     = struct {
		UserNewServiceHits        int        `json:"user_new_service_hits"`
		UserMonolithFallbackHits  int        `json:"user_monolith_fallback_hits"`
		OrderNewServiceHits       int        `json:"order_new_service_hits"`
		OrderMonolithFallbackHits int        `json:"order_monolith_fallback_hits"`
		TotalRequests             int        `json:"total_requests"`
		RecentLogs                []LogEntry `json:"recent_logs"`
	}{
		RecentLogs: make([]LogEntry, 0),
	}
)

func main() {
	// 0. Load Configuration
	// API Gateway is at the root level relative to services, but let's be safe.
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found in parent directory, trying local .env")
		godotenv.Load(".env")
	}

	monolithAddr := os.Getenv("MONOLITH_URL")
	if monolithAddr == "" {
		log.Fatal("MONOLITH_URL is not set")
	}
	userAddr := os.Getenv("USER_SERVICE_URL")
	if userAddr == "" {
		log.Fatal("USER_SERVICE_URL is not set")
	}
	orderAddr := os.Getenv("ORDER_SERVICE_URL")
	if orderAddr == "" {
		log.Fatal("ORDER_SERVICE_URL is not set")
	}

	monolithURL, _ := url.Parse(monolithAddr)
	userServiceURL, _ := url.Parse(userAddr)
	orderServiceURL, _ := url.Parse(orderAddr)

	monolithProxy := httputil.NewSingleHostReverseProxy(monolithURL)
	userServiceProxy := httputil.NewSingleHostReverseProxy(userServiceURL)
	orderServiceProxy := httputil.NewSingleHostReverseProxy(orderServiceURL)

	r := gin.Default()
	r.LoadHTMLFiles("dashboard.html")

	// 1. Dashboard UI
	r.GET("/dashboard", func(c *gin.Context) {
		metricsLock.Lock()
		defer metricsLock.Unlock()

		userTotal := metrics.UserNewServiceHits + metrics.UserMonolithFallbackHits
		orderTotal := metrics.OrderNewServiceHits + metrics.OrderMonolithFallbackHits

		userSuccessRate := 0
		if userTotal > 0 {
			userSuccessRate = (metrics.UserNewServiceHits * 100) / userTotal
		}

		orderSuccessRate := 0
		if orderTotal > 0 {
			orderSuccessRate = (metrics.OrderNewServiceHits * 100) / orderTotal
		}

		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"TotalRequests":     metrics.TotalRequests,
			"UserTotal":         userTotal,
			"UserNewHits":       metrics.UserNewServiceHits,
			"UserFallbackHits":  metrics.UserMonolithFallbackHits,
			"UserSuccessRate":   userSuccessRate,
			"OrderTotal":        orderTotal,
			"OrderNewHits":      metrics.OrderNewServiceHits,
			"OrderFallbackHits": metrics.OrderMonolithFallbackHits,
			"OrderSuccessRate":  orderSuccessRate,
			"RecentLogs":        metrics.RecentLogs,
		})
	})

	r.GET("/metrics", func(c *gin.Context) {
		metricsLock.Lock()
		defer metricsLock.Unlock()
		c.JSON(http.StatusOK, metrics)
	})

	// 2. Strangler Routing Logic
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		metricsLock.Lock()
		metrics.TotalRequests++
		metricsLock.Unlock()

		// Rule 1: All Writes (POST, PUT, DELETE) go to the Monolith
		if method != http.MethodGet {
			log.Printf("[PROXY] Write operation: %s %s -> Monolith", method, path)
			monolithProxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Rule 2: GET /users/:id -> User Service with Fallback
		if strings.HasPrefix(path, "/users/") {
			log.Printf("[PROXY] GET User: %s -> Attempting User Service", path)
			proxyWithFallback(c, userServiceProxy, monolithProxy, "user")
			return
		}

		// Rule 3: GET /orders/:id -> Order Service with Fallback
		if strings.HasPrefix(path, "/orders/") {
			log.Printf("[PROXY] GET Order: %s -> Attempting Order Service", path)
			proxyWithFallback(c, orderServiceProxy, monolithProxy, "order")
			return
		}

		// Rule 4: Everything else -> Monolith
		log.Printf("[PROXY] Default: %s %s -> Monolith", method, path)
		monolithProxy.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway"})
	})

	log.Println("API Gateway (Strangler) starting on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Could not start API Gateway: %v", err)
	}
}

// proxyWithFallback attempts to use the primary proxy, but falls back to monolith if 404
func proxyWithFallback(c *gin.Context, primary *httputil.ReverseProxy, fallback *httputil.ReverseProxy, serviceType string) {
	writer := &statusSyncWriter{ResponseWriter: c.Writer, status: http.StatusOK}
	primary.ServeHTTP(writer, c.Request)

	if writer.status == http.StatusNotFound {
		log.Printf("[FALLBACK] 404 from Primary Service. Redirecting %s to Monolith", c.Request.URL.Path)

		metricsLock.Lock()
		if serviceType == "user" {
			metrics.UserMonolithFallbackHits++
		} else {
			metrics.OrderMonolithFallbackHits++
		}
		metricsLock.Unlock()

		// CRITICAL: We must reset any headers set by the first proxy attempt (like Content-Length)
		// before invoking the second proxy, otherwise curl gets malformed headers.
		for k := range c.Writer.Header() {
			c.Writer.Header().Del(k)
		}
		fallback.ServeHTTP(c.Writer, c.Request)
	} else if writer.status == http.StatusOK {
		metricsLock.Lock()
		if serviceType == "user" {
			metrics.UserNewServiceHits++
		} else {
			metrics.OrderNewServiceHits++
		}
		metricsLock.Unlock()
	}

	// Add to log
	metricsLock.Lock()
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		ServiceID: strings.Title(serviceType),
		ID:        strings.Split(c.Request.URL.Path, "/")[len(strings.Split(c.Request.URL.Path, "/"))-1],
		Result:    func() string { if writer.status == http.StatusNotFound { return "Monolith" } else { return "Microservice" } }(),
	}
	metrics.RecentLogs = append([]LogEntry{entry}, metrics.RecentLogs...)
	if len(metrics.RecentLogs) > 10 {
		metrics.RecentLogs = metrics.RecentLogs[:10]
	}
	metricsLock.Unlock()
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
