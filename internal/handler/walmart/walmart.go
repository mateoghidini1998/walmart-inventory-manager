package walmart

import (
	"encoding/json"
	"net/http"
	"walmart-inventory-manager/internal/walmart"
)

type TokenHandler struct {
	walmartClient *walmart.Client
}

func NewTokenHandler(client *walmart.Client) *TokenHandler {
	return &TokenHandler{walmartClient: client}
}

func (h *TokenHandler) GetToken(w http.ResponseWriter, r *http.Request) (err error) {
	token, expires_in, err := h.walmartClient.GetAccessToken()
	if err != nil {
		http.Error(w, "Failed to get token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"access_token": token,
		"expires_in": expires_in,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return
}
