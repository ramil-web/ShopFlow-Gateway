package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"shopflow/gateway/middleware"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Auth routes проксируем напрямую на Auth сервис
	r.Any("/auth/*proxyPath", proxyHandler(os.Getenv("AUTH_URL")))

	// Application routes защищенные JWT
	r.Any("/applications/*proxyPath", middleware.AuthMiddleware(), proxyHandler(os.Getenv("APPLICATION_URL")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	fmt.Println("Gateway running on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// proxyHandler проксирует запрос к сервису targetURL
func proxyHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if targetURL == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "services URL not set"})
			return
		}

		client := &http.Client{}
		url := strings.TrimRight(targetURL, "/") + c.Request.RequestURI

		// Читаем тело запроса
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		req, _ := http.NewRequest(c.Request.Method, url, bytes.NewBuffer(bodyBytes))
		req.Header = c.Request.Header

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
	}
}
