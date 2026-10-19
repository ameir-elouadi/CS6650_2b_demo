package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
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
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
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
