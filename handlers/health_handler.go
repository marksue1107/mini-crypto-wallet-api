package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/db_conn"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck 健康檢查端點
//
// @Summary Health check
// @Description Check if the service is healthy
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "mini-crypto-wallet-api",
	})
}

// ReadinessCheck 就緒檢查端點
//
// @Summary Readiness check
// @Description Check if the service is ready to serve requests
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// 檢查資料庫連接
	sqlDB, err := db_conn.Conn_DB.MasterDB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"service": "mini-crypto-wallet-api",
	})
}
