package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// apiKeyAuthMiddleware API Key 认证中间件
func apiKeyAuthMiddleware() gin.HandlerFunc {
	apiKey := os.Getenv("API_KEY")

	return func(c *gin.Context) {
		// 如果没有配置 API_KEY，跳过认证
		if apiKey == "" {
			c.Next()
			return
		}

		key := c.GetHeader("X-API-Key")
		if key == "" {
			respondError(c, http.StatusUnauthorized, "MISSING_API_KEY",
				"缺少 API Key", "请在请求头中设置 X-API-Key")
			c.Abort()
			return
		}

		if key != apiKey {
			respondError(c, http.StatusUnauthorized, "INVALID_API_KEY",
				"API Key 无效", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// errorHandlingMiddleware 错误处理中间件
func errorHandlingMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logrus.Errorf("服务器内部错误: %v, path: %s", recovered, c.Request.URL.Path)

		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR",
			"服务器内部错误", recovered)
	})
}
