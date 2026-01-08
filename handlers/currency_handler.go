package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/services"
)

type CurrencyHandler struct {
	service *services.CurrencyService
}

func NewCurrencyHandler(service *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{service}
}

// GetCurrencies 獲取所有幣種列表
//
// @Summary Get all currencies
// @Description Get list of all active currencies
// @Tags Currency
// @Produce json
// @Success 200 {array} models.Currency
// @Router /currencies [get]
func (h *CurrencyHandler) GetCurrencies(c *gin.Context) {
	currencies, err := h.service.GetAllCurrencies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch currencies"})
		return
	}
	c.JSON(http.StatusOK, currencies)
}

// GetCurrency 根據 ID 獲取幣種
//
// @Summary Get currency by ID
// @Description Get currency details by ID
// @Tags Currency
// @Produce json
// @Param id path int true "Currency ID"
// @Success 200 {object} models.Currency
// @Failure 404 {object} map[string]string
// @Router /currencies/{id} [get]
func (h *CurrencyHandler) GetCurrency(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid currency id"})
		return
	}

	currency, err := h.service.GetCurrencyByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "currency not found"})
		return
	}
	c.JSON(http.StatusOK, currency)
}
