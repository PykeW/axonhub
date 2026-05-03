package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/looplj/axonhub/internal/objects"
)

func TestRelaySubKeyRESTContract(t *testing.T) {
	got := RelaySubKeyRESTContract()
	want := []RelaySubKeyEndpoint{
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/products"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/products/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/products"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/products/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/product-channels"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/product-channels/:id"},
		{Method: http.MethodDelete, Path: "/admin/relay-subkeys/product-channels/:id"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/keys"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/keys/:id/status"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/keys/:id/limits"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id/wallet"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id/ledger"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/wallets/recharge"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/requests"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/channel-pool-health"},
		{Method: http.MethodGet, Path: "/admin/share-use/wallet"},
		{Method: http.MethodGet, Path: "/admin/share-use/ledger"},
		{Method: http.MethodGet, Path: "/admin/share-use/usage"},
		{Method: http.MethodGet, Path: "/admin/projects/:projectId/relay-subkeys/overview"},
		{Method: http.MethodGet, Path: "/admin/projects/:projectId/relay-subkeys/usage"},
	}

	if len(got) != len(want) {

		t.Fatalf("RelaySubKeyRESTContract length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("RelaySubKeyRESTContract[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRelaySubKeyHandlersRejectInvalidIDsAndBadJSONWithErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlers := &RelaySubKeyHandlers{}

	tests := []struct {
		name        string
		method      string
		path        string
		body        string
		register    func(*gin.Engine)
		wantMessage string
	}{
		{
			name:   "path ID must be numeric or GUID",
			method: http.MethodGet,
			path:   "/admin/relay-subkeys/keys/not-a-guid",
			register: func(router *gin.Engine) {
				router.GET("/admin/relay-subkeys/keys/:id", handlers.GetKey)
			},
			wantMessage: "id must be a numeric id or AxonHub GUID",
		},
		{
			name:   "project query ID must be positive",
			method: http.MethodGet,
			path:   "/admin/relay-subkeys/keys?projectId=0",
			register: func(router *gin.Engine) {
				router.GET("/admin/relay-subkeys/keys", handlers.ListKeys)
			},
			wantMessage: "projectId must be a numeric id or AxonHub GUID",
		},
		{
			name:   "malformed JSON keeps the standard error envelope",
			method: http.MethodPost,
			path:   "/admin/relay-subkeys/products",
			body:   "{",
			register: func(router *gin.Engine) {
				router.POST("/admin/relay-subkeys/products", handlers.CreateProduct)
			},
			wantMessage: "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			tt.register(router)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			router.ServeHTTP(recorder, req)
			assertRelaySubKeyErrorResponse(t, recorder, http.StatusBadRequest, tt.wantMessage)
		})
	}
}

func assertRelaySubKeyErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, wantMessage string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, status, recorder.Body.String())
	}

	var payload objects.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response body is not objects.ErrorResponse JSON: %v; body=%s", err, recorder.Body.String())
	}
	if payload.Error.Type != http.StatusText(status) {
		t.Fatalf("error.type = %q, want %q", payload.Error.Type, http.StatusText(status))
	}
	if !strings.Contains(payload.Error.Message, wantMessage) {
		t.Fatalf("error.message = %q, want to contain %q", payload.Error.Message, wantMessage)
	}
}
