package controller

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"

	"node-go-labs/go-service/internal/httpjson"
	"node-go-labs/go-service/internal/service"
)

type AccountController struct {
	service *service.AccountService
}

func NewAccountController(accountService *service.AccountService) *AccountController {
	return &AccountController{service: accountService}
}

func (c *AccountController) List(w http.ResponseWriter, _ *http.Request) {
	accounts, err := c.service.List()
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusOK, accounts)
}

func (c *AccountController) Transfer(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FromAccountID int     `json:"from_account_id"`
		ToAccountID   int     `json:"to_account_id"`
		Amount        float64 `json:"amount"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body contains invalid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object")
		return
	}
	if input.FromAccountID < 1 || input.ToAccountID < 1 || input.Amount <= 0 || math.Round(input.Amount*100) != input.Amount*100 {
		httpjson.WriteError(w, http.StatusBadRequest, "validation_error", "Account ids must be positive and amount must have at most two decimals")
		return
	}
	accounts, err := c.service.Transfer(input.FromAccountID, input.ToAccountID, input.Amount)
	if err != nil {
		var accountError *service.AccountError
		if errors.As(err, &accountError) {
			status := http.StatusBadRequest
			if accountError.Code == "account_not_found" {
				status = http.StatusNotFound
			} else if accountError.Code == "insufficient_balance" {
				status = http.StatusConflict
			}
			httpjson.WriteError(w, status, accountError.Code, accountError.Message)
			return
		}
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"status": "committed", "accounts": accounts})
}
