package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"camper-vane/internal/db"
)

func TestUserConfigAPI(t *testing.T) {
	repo, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	userHandler := NewUserHandler(repo)
	authHandler := NewAuthHandler(repo)

	// Obtain session cookie
	wAuth := httptest.NewRecorder()
	reqAuth := httptest.NewRequest("GET", "/api/v1/auth/callback?code=testcode&mock_user_id=user-cfg-test", nil)
	authHandler.HandleCallback(wAuth, reqAuth)
	cookie := wAuth.Result().Cookies()[0]

	// Test 1: Reject negative daily token cap
	negBody, _ := json.Marshal(UpdateUserConfigRequest{
		DailyTokenCap:   -500,
		RoutingStrategy: "simple",
	})
	reqNeg := httptest.NewRequest("PUT", "/api/v1/user/config", bytes.NewBuffer(negBody))
	reqNeg.AddCookie(cookie)
	wNeg := httptest.NewRecorder()

	authHandler.RequireAuth(userHandler.HandleUserConfig)(wNeg, reqNeg)
	if wNeg.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for negative cap, got %d", wNeg.Result().StatusCode)
	}

	// Test 2: Reject invalid routing strategy
	stratBody, _ := json.Marshal(UpdateUserConfigRequest{
		DailyTokenCap:   10000,
		RoutingStrategy: "invalid_strategy",
	})
	reqStrat := httptest.NewRequest("PUT", "/api/v1/user/config", bytes.NewBuffer(stratBody))
	reqStrat.AddCookie(cookie)
	wStrat := httptest.NewRecorder()

	authHandler.RequireAuth(userHandler.HandleUserConfig)(wStrat, reqStrat)
	if wStrat.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid strategy, got %d", wStrat.Result().StatusCode)
	}

	// Test 3: Valid update
	validBody, _ := json.Marshal(UpdateUserConfigRequest{
		DailyTokenCap:   75000,
		RoutingStrategy: "advanced",
		PreferredModels: []string{"claude-3-5-sonnet"},
	})
	reqValid := httptest.NewRequest("PUT", "/api/v1/user/config", bytes.NewBuffer(validBody))
	reqValid.AddCookie(cookie)
	wValid := httptest.NewRecorder()

	authHandler.RequireAuth(userHandler.HandleUserConfig)(wValid, reqValid)
	if wValid.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for valid update, got %d", wValid.Result().StatusCode)
	}

	// Verify via GET
	reqGet := httptest.NewRequest("GET", "/api/v1/user/config", nil)
	reqGet.AddCookie(cookie)
	wGet := httptest.NewRecorder()
	authHandler.RequireAuth(userHandler.HandleUserConfig)(wGet, reqGet)

	var updatedCfg db.UserConfig
	_ = json.NewDecoder(wGet.Result().Body).Decode(&updatedCfg)
	if updatedCfg.DailyTokenCap != 75000 || updatedCfg.RoutingStrategy != "advanced" {
		t.Errorf("unexpected updated config: %+v", updatedCfg)
	}
}
