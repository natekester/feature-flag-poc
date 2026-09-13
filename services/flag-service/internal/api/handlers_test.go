package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flag-service/internal/api"
	"flag-service/internal/domain"
)

type closeNotifierRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func (c *closeNotifierRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func newCloseNotifierRecorder() *closeNotifierRecorder {
	return &closeNotifierRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closed:           make(chan bool, 1),
	}
}

func setupTestRouter() (*gin.Engine, *api.FlagHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := api.NewFlagHandler(nil, "FeatureFlags", nil)

	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/admin/flags", handler.ListFlags)
		apiGroup.PUT("/admin/flags/:key/overrides/users/:userId", handler.SetUserOverride)
		apiGroup.DELETE("/admin/flags/:key/overrides/users/:userId", handler.RemoveUserOverride)
		apiGroup.GET("/evaluate", handler.EvaluateUser)
		apiGroup.GET("/stream", handler.SSEStream)
	}

	return router, handler
}

func TestListFlags_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/admin/flags", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var flags []domain.FeatureFlag
	err = json.Unmarshal(w.Body.Bytes(), &flags)
	require.NoError(t, err)
	assert.NotEmpty(t, flags)
	assert.Equal(t, "ds-button-v2", flags[0].Key)
	assert.True(t, flags[0].Enabled)
}

func TestSetUserOverride_Success(t *testing.T) {
	router, _ := setupTestRouter()

	payload := map[string]any{
		"variation": "v2",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, "/api/v1/admin/flags/ds-button-v2/overrides/users/user_42", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, "ds-button-v2", resp["flagKey"])
	assert.Equal(t, "user_42", resp["userId"])
	assert.Equal(t, "v2", resp["variation"])
}

func TestSetUserOverride_BadRequest(t *testing.T) {
	router, _ := setupTestRouter()

	// Invalid JSON payload
	req, err := http.NewRequest(http.MethodPut, "/api/v1/admin/flags/ds-button-v2/overrides/users/user_42", bytes.NewBufferString("{invalid}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveUserOverride_Success(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. Set override
	payload := map[string]any{"variation": "v2"}
	body, _ := json.Marshal(payload)
	req1, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/flags/ds-button-v2/overrides/users/user_42", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. Remove override
	req2, err := http.NewRequest(http.MethodDelete, "/api/v1/admin/flags/ds-button-v2/overrides/users/user_42", nil)
	require.NoError(t, err)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, "ds-button-v2", resp["flagKey"])
	assert.Equal(t, "user_42", resp["userId"])
}

func TestEvaluateUser_Success(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. First evaluate before override
	req1, err := http.NewRequest(http.MethodGet, "/api/v1/evaluate?userId=user_999", nil)
	require.NoError(t, err)

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.NotEmpty(t, w1.Header().Get("ETag"))

	// 2. Set explicit override for user_999 -> compact
	overridePayload, _ := json.Marshal(map[string]any{"variation": "compact"})
	reqOverride, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/flags/ds-button-v2/overrides/users/user_999", bytes.NewBuffer(overridePayload))
	reqOverride.Header.Set("Content-Type", "application/json")
	wOverride := httptest.NewRecorder()
	router.ServeHTTP(wOverride, reqOverride)
	assert.Equal(t, http.StatusOK, wOverride.Code)

	// 3. Re-evaluate to verify user_999 gets compact variant
	req2, err := http.NewRequest(http.MethodGet, "/api/v1/evaluate?userId=user_999", nil)
	require.NoError(t, err)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var evaluated map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &evaluated)
	require.NoError(t, err)
	assert.Equal(t, "compact", evaluated["ds-button-v2"])
}

func TestEvaluateUser_MissingUserId(t *testing.T) {
	router, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/evaluate", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSSEStream_Headers(t *testing.T) {
	router, _ := setupTestRouter()

	req, err := http.NewRequest(http.MethodGet, "/api/v1/stream", nil)
	require.NoError(t, err)

	w := newCloseNotifierRecorder()

	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	w.closed <- true

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache, no-transform", w.Header().Get("Cache-Control"))
}
