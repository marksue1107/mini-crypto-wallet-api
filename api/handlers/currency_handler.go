package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apierrors "mini-crypto-wallet-api/internal/errors"
	"mini-crypto-wallet-api/models"
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
// @Success 200 {array} models.CurrencyResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /currencies [get]
func (h *CurrencyHandler) GetCurrencies(c *gin.Context) {
	currencies, err := h.service.GetAllCurrencies()
	if err != nil {
		apierrors.RespondError(c, http.StatusInternalServerError, apierrors.ErrCodeInternalError, errFailedToFetchCurrencies)
		return
	}

	// Convert models to DTOs
	responses := models.ToCurrencyResponses(currencies)
	c.JSON(http.StatusOK, responses)
}

// GetCurrency 根據 ID 獲取幣種
//
// @Summary Get currency by ID
// @Description Get currency details by ID
// @Tags Currency
// @Produce json
// @Param id path int true "Currency ID"
// @Success 200 {object} models.CurrencyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /currencies/{id} [get]
func (h *CurrencyHandler) GetCurrency(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apierrors.RespondError(c, http.StatusBadRequest, apierrors.ErrCodeInvalidRequest, errInvalidCurrencyID)
		return
	}

	currency, err := h.service.GetCurrencyByID(uint(id))
	if err != nil {
		apierrors.RespondError(c, http.StatusNotFound, apierrors.ErrCodeNotFound, errCurrencyNotFound)
		return
	}

	// Convert model to DTO
	response := models.ToCurrencyResponse(currency)
	c.JSON(http.StatusOK, response)
}
