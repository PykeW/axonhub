package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/scopes"
)

type relayAuthzContextFunc func(context.Context) context.Context

func TestRequireScopesRejectsMissingPrincipal(t *testing.T) {
	recorder, handled := performRelaySystemAuthzRecorder(t, nil)
	assertRelayAuthzErrorResponse(t, recorder, http.StatusUnauthorized, "authentication required")
	if handled {
		t.Fatal("handler should not run without authentication")
	}
}

func TestRequireScopesRejectsOrdinaryAPIKeyPrincipal(t *testing.T) {
	recorder, handled := performRelaySystemAuthzRecorder(t, func(ctx context.Context) context.Context {
		ctx = authz.NewAPIKeyContext(ctx, 10, 20)
		return contexts.WithAPIKey(ctx, &ent.APIKey{
			ID:     10,
			Scopes: []string{string(scopes.ScopeReadAPIKeys)},
		})
	})
	assertRelayAuthzErrorResponse(t, recorder, http.StatusUnauthorized, "user authentication required")
	if handled {
		t.Fatal("handler should not run for ordinary API key authentication")
	}
}

func TestRequireScopesRejectsUserWithoutRequiredScope(t *testing.T) {
	user := &ent.User{ID: 1, Scopes: []string{string(scopes.ScopeReadRequests)}}
	recorder, handled := performRelaySystemAuthzRecorder(t, relayUserContext(user))
	assertRelayAuthzErrorResponse(t, recorder, http.StatusForbidden, "missing required scope read_api_keys")
	if handled {
		t.Fatal("handler should not run without required system scope")
	}
}

func TestRequireScopesAllowsUserWithRequiredScope(t *testing.T) {
	user := &ent.User{ID: 1, Scopes: []string{string(scopes.ScopeReadAPIKeys)}}
	code, handled := performRelaySystemAuthzRequest(t, relayUserContext(user))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if !handled {
		t.Fatal("handler should run for authorized user")
	}
}

func TestRequireScopesAllowsOwnerWithoutExplicitScope(t *testing.T) {
	user := &ent.User{ID: 1, IsOwner: true}
	code, handled := performRelaySystemAuthzRequest(t, relayUserContext(user))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if !handled {
		t.Fatal("handler should run for owner user")
	}
}

func TestRequireProjectScopesRejectsOrdinaryAPIKeyPrincipal(t *testing.T) {
	recorder, _, handled := performRelayProjectAuthzRecorder(t, func(ctx context.Context) context.Context {
		ctx = authz.NewAPIKeyContext(ctx, 10, 20)
		return contexts.WithAPIKey(ctx, &ent.APIKey{
			ID:        10,
			ProjectID: 7,
			Scopes:    []string{string(scopes.ScopeReadAPIKeys), string(scopes.ScopeReadRequests)},
		})
	}, "/admin/projects/7/relay-subkeys/overview")
	assertRelayAuthzErrorResponse(t, recorder, http.StatusUnauthorized, "user authentication required")
	if handled {
		t.Fatal("handler should not run for ordinary API key authentication")
	}
}

func TestRequireProjectScopesRejectsCrossProjectAccess(t *testing.T) {
	user := &ent.User{
		ID: 1,
		Edges: ent.UserEdges{
			ProjectUsers: []*ent.UserProject{{
				ProjectID: 1,
				Scopes:    []string{string(scopes.ScopeReadAPIKeys), string(scopes.ScopeReadRequests)},
			}},
		},
	}

	recorder, projectID, handled := performRelayProjectAuthzRecorder(t, relayUserContext(user), "/admin/projects/2/relay-subkeys/overview")
	assertRelayAuthzErrorResponse(t, recorder, http.StatusForbidden, "project 2 access denied")
	if handled {
		t.Fatalf("handler should not run for cross-project access; saw project %d", projectID)
	}
}

func TestRequireProjectScopesAllowsUserWithSystemScopes(t *testing.T) {
	user := &ent.User{
		ID:     1,
		Scopes: []string{string(scopes.ScopeReadAPIKeys), string(scopes.ScopeReadRequests)},
	}

	code, projectID, handled := performRelayProjectAuthzRequest(t, relayUserContext(user), "/admin/projects/7/relay-subkeys/overview")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if !handled {
		t.Fatal("handler should run for authorized system-scope user")
	}
	if projectID != 7 {
		t.Fatalf("project id in context = %d, want 7", projectID)
	}
}

func TestRequireProjectScopesAllowsProjectMemberWithScopes(t *testing.T) {
	user := &ent.User{
		ID: 1,
		Edges: ent.UserEdges{
			ProjectUsers: []*ent.UserProject{{
				ProjectID: 7,
				Scopes:    []string{string(scopes.ScopeReadAPIKeys), string(scopes.ScopeReadRequests)},
			}},
		},
	}

	code, projectID, handled := performRelayProjectAuthzRequest(t, relayUserContext(user), "/admin/projects/7/relay-subkeys/overview")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if !handled {
		t.Fatal("handler should run for authorized project member")
	}
	if projectID != 7 {
		t.Fatalf("project id in context = %d, want 7", projectID)
	}
}

func TestRequireProjectScopesAllowsProjectOwnerWithoutScopes(t *testing.T) {
	user := &ent.User{
		ID: 1,
		Edges: ent.UserEdges{
			ProjectUsers: []*ent.UserProject{{ProjectID: 7, IsOwner: true}},
		},
	}

	code, projectID, handled := performRelayProjectAuthzRequest(t, relayUserContext(user), "/admin/projects/7/relay-subkeys/usage")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if !handled {
		t.Fatal("handler should run for project owner")
	}
	if projectID != 7 {
		t.Fatalf("project id in context = %d, want 7", projectID)
	}
}

func TestRequireProjectScopesRejectsMissingRequiredScope(t *testing.T) {
	user := &ent.User{
		ID: 1,
		Edges: ent.UserEdges{
			ProjectUsers: []*ent.UserProject{{
				ProjectID: 7,
				Scopes:    []string{string(scopes.ScopeReadRequests)},
			}},
		},
	}

	recorder, _, handled := performRelayProjectAuthzRecorder(t, relayUserContext(user), "/admin/projects/7/relay-subkeys/overview")
	assertRelayAuthzErrorResponse(t, recorder, http.StatusForbidden, "project 7 access denied")
	if handled {
		t.Fatal("handler should not run without all required project scopes")
	}
}

func TestRequireProjectScopesRejectsInvalidProjectIDWithErrorShape(t *testing.T) {
	user := &ent.User{ID: 1, Scopes: []string{string(scopes.ScopeReadAPIKeys), string(scopes.ScopeReadRequests)}}
	recorder, _, handled := performRelayProjectAuthzRecorder(t, relayUserContext(user), "/admin/projects/not-a-project/relay-subkeys/overview")
	assertRelayAuthzErrorResponse(t, recorder, http.StatusBadRequest, "project ID must be a numeric id or AxonHub project GUID")
	if handled {
		t.Fatal("handler should not run with invalid project id")
	}
}

func relayUserContext(user *ent.User) relayAuthzContextFunc {
	return func(ctx context.Context) context.Context {
		ctx = authz.NewUserContext(ctx, user.ID)
		return contexts.WithUser(ctx, user)
	}
}

func performRelaySystemAuthzRequest(t *testing.T, withContext relayAuthzContextFunc) (int, bool) {
	t.Helper()
	recorder, handled := performRelaySystemAuthzRecorder(t, withContext)
	return recorder.Code, handled
}

func performRelaySystemAuthzRecorder(t *testing.T, withContext relayAuthzContextFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	handled := false
	router := gin.New()
	router.GET(
		"/admin/relay-subkeys/keys",
		injectRelayAuthzContext(withContext),
		RequireScopes(scopes.ScopeReadAPIKeys),
		func(c *gin.Context) {
			handled = true
			c.Status(http.StatusOK)
		},
	)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/relay-subkeys/keys", nil)
	router.ServeHTTP(recorder, req)
	return recorder, handled
}

func performRelayProjectAuthzRequest(t *testing.T, withContext relayAuthzContextFunc, path string) (int, int, bool) {
	t.Helper()
	recorder, projectID, handled := performRelayProjectAuthzRecorder(t, withContext, path)
	return recorder.Code, projectID, handled
}

func performRelayProjectAuthzRecorder(t *testing.T, withContext relayAuthzContextFunc, path string) (*httptest.ResponseRecorder, int, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	projectID := 0
	handled := false
	router := gin.New()
	router.GET(
		"/admin/projects/:projectId/relay-subkeys/:view",
		injectRelayAuthzContext(withContext),
		RequireProjectScopes("projectId", scopes.ScopeReadAPIKeys, scopes.ScopeReadRequests),
		func(c *gin.Context) {
			handled = true
			projectID, _ = contexts.GetProjectID(c.Request.Context())
			c.Status(http.StatusOK)
		},
	)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, req)
	return recorder, projectID, handled
}

func assertRelayAuthzErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, wantMessage string) {
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

func injectRelayAuthzContext(withContext relayAuthzContextFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if withContext != nil {
			c.Request = c.Request.WithContext(withContext(c.Request.Context()))
		}
		c.Next()
	}
}
