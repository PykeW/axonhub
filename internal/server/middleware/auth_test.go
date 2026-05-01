package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/server/biz"
)

func TestWithAPIKeyConfig_RejectsNoAuthKeyWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(WithAPIKeyConfig(&biz.AuthService{}, nil))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+biz.NoAuthAPIKeyValue)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestWithAPIKeyConfig_AllowsMissingAuthorizationWhenNoAuthAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		key, err := ExtractAPIKeyFromRequest(c.Request, &APIKeyConfig{
			Headers:       []string{"Authorization"},
			RequireBearer: true,
		})
		if errors.Is(err, ErrAPIKeyRequired) {
			c.Status(http.StatusNoContent)
			c.Abort()

			return
		}

		if err != nil || key != "" {
			c.Status(http.StatusTeapot)
			c.Abort()

			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

type relayAuthResolverFunc func(context.Context, *ent.APIKey) (*biz.RelayAuthContext, error)

func (f relayAuthResolverFunc) ResolveRelayAuth(ctx context.Context, apiKey *ent.APIKey) (*biz.RelayAuthContext, error) {
	return f(ctx, apiKey)
}

func TestReleaseRelayInflight(t *testing.T) {
	tracker := biz.NewRelayInflightTracker()
	svc := biz.NewRelayRuntimeService()
	svc.SetInflightTracker(tracker)
	limit := int64(1)
	svc.SetResolver(relayAuthResolverFunc(func(ctx context.Context, apiKey *ent.APIKey) (*biz.RelayAuthContext, error) {
		return &biz.RelayAuthContext{RelayKeyID: 9001, Quota: biz.RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}, nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 1, ProjectID: 2})
	if err != nil || decision == nil || !decision.Allowed || relay == nil {
		t.Fatalf("expected allowed relay, decision=%v relay=%v err=%v", decision, relay, err)
	}
	if got := tracker.Current(relay.RelayKeyID); got != 1 {
		t.Fatalf("expected inflight begin, got %d", got)
	}

	ctx := contexts.WithRelayAuthContext(context.Background(), relay)
	releaseRelayInflight(ctx)
	releaseRelayInflight(ctx)
	if got := tracker.Current(relay.RelayKeyID); got != 0 {
		t.Fatalf("expected middleware release to clear inflight once, got %d", got)
	}
}


