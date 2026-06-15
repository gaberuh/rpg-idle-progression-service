package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler godoc
// @Summary     Health check
// @Tags        health
// @Produce     json
// @Success     200
// @Router      /health [get]
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
