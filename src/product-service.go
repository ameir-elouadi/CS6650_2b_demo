package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Models

type Product struct {
	ProductID    *int32  `json:"product_id" binding:"required,min=1"`
	SKU          *string `json:"sku" binding:"required,min=1,max=100"`
	Manufacturer *string `json:"manufacturer" binding:"required,min=1,max=200"`
	CategoryID   *int32  `json:"category_id" binding:"required,min=1"`
	Weight       *int32  `json:"weight" binding:"required,min=0"`
	SomeOtherID  *int32  `json:"some_other_id" binding:"required,min=1"`
}

type Error struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Store

var (
	products = map[int32]Product{}
	mu       sync.RWMutex
)

// Observability: RED signals (rate, errors, duration) per endpoint

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Requests handled, by endpoint and status code.",
	}, []string{"method", "endpoint", "status"})

	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Requests that returned a non-2xx status, by endpoint and status code.",
	}, []string{"method", "endpoint", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Time spent handling a request, by endpoint.",
		// 50µs to ~1.6s: requests take well under the default buckets' 5ms minimum
		Buckets: prometheus.ExponentialBuckets(0.00005, 2, 16),
	}, []string{"method", "endpoint"})

	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

// observe records metrics and writes one JSON log line per request.
func observe(c *gin.Context) {
	start := time.Now()
	c.Next()
	duration := time.Since(start)

	// Route template (e.g. /products/:productId), not the raw path, so IDs don't explode label cardinality
	endpoint := c.FullPath()
	if endpoint == "" {
		endpoint = "unmatched"
	}
	method := c.Request.Method
	status := c.Writer.Status()
	statusLabel := strconv.Itoa(status)

	requestsTotal.WithLabelValues(method, endpoint, statusLabel).Inc()
	if status < 200 || status > 299 {
		errorsTotal.WithLabelValues(method, endpoint, statusLabel).Inc()
	}
	requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	attrs := []any{
		"method", method,
		"endpoint", endpoint,
		"path", c.Request.URL.Path,
		"status", status,
		"duration_ms", float64(duration.Microseconds()) / 1000,
	}
	if p, ok := c.Get("panic"); ok {
		attrs = append(attrs, "error", p)
	}
	logger.Info("request", attrs...)
}

// Helpers

func parseProductID(c *gin.Context) (int32, bool) {
	id, err := strconv.ParseInt(c.Param("productId"), 10, 32)
	if err != nil || id < 1 {
		c.JSON(http.StatusNotFound, Error{"NOT_FOUND", "Product not found", "product ID must be a positive 32-bit integer"})
		return 0, false
	}
	return int32(id), true
}

// Handlers

func getProduct(c *gin.Context) {
	id, ok := parseProductID(c)
	if !ok {
		return
	}

	mu.RLock()
	product, found := products[id]
	mu.RUnlock()

	if !found {
		c.JSON(http.StatusNotFound, Error{"NOT_FOUND", "Product not found", ""})
		return
	}
	c.JSON(http.StatusOK, product)
}

func addProductDetails(c *gin.Context) {
	id, ok := parseProductID(c)
	if !ok {
		return
	}

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, Error{"INVALID_INPUT", "The provided input data is invalid", err.Error()})
		return
	}
	if *product.ProductID != id {
		c.JSON(http.StatusBadRequest, Error{"INVALID_INPUT", "The provided input data is invalid", "product_id in body must match product ID in path"})
		return
	}

	mu.Lock()
	products[id] = product
	mu.Unlock()

	c.Status(http.StatusNoContent)
}

// Main

func main() {
	// Release mode keeps Gin's debug lines out of stdout, so every log line is JSON
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Scraped by Prometheus; registered before the middleware so scrapes aren't counted as traffic
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.Use(observe)
	// The panic goes into the request's JSON log line instead of a stack trace
	r.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, err any) {
		c.Set("panic", fmt.Sprint(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, Error{"INTERNAL_ERROR", "Internal server error", ""})
	}))
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Simulate-Error") == "true" {
			panic("simulated error")
		}
	})

	r.GET("/products/:productId", getProduct)
	r.POST("/products/:productId/details", addProductDetails)

	r.Run(":8080")
}
