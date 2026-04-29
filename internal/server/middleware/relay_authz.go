package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/scopes"
)

// RequireScopes enforces system-level scope checks for authenticated admin routes.
func RequireScopes(requiredScopes ...scopes.ScopeSlug) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireUserPrincipal(c) {
			return
		}

		for _, requiredScope := range requiredScopes {
			if !authz.HasScope(c.Request.Context(), requiredScope) {
				AbortWithError(c, http.StatusForbidden, fmt.Errorf("missing required scope %s", requiredScope))
				return
			}
		}

		c.Next()
	}
}

// RequireProjectScopes enforces project membership plus the supplied project scopes.
func RequireProjectScopes(projectParam string, requiredScopes ...scopes.ScopeSlug) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireUserPrincipal(c) {
			return
		}

		projectID, ok := projectIDFromParam(c, projectParam)
		if !ok {
			return
		}

		user, ok := contexts.GetUser(c.Request.Context())
		if !ok || user == nil {
			AbortWithError(c, http.StatusUnauthorized, errors.New("user authentication required"))
			return
		}

		if !userCanAccessProjectWithScopes(user, projectID, requiredScopes...) {
			AbortWithError(c, http.StatusForbidden, fmt.Errorf("project %d access denied", projectID))
			return
		}

		ctx := contexts.WithProjectID(c.Request.Context(), projectID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func requireUserPrincipal(c *gin.Context) bool {
	principal, ok := authz.GetPrincipal(c.Request.Context())
	if !ok {
		AbortWithError(c, http.StatusUnauthorized, errors.New("authentication required"))
		return false
	}
	if !principal.IsUser() {
		AbortWithError(c, http.StatusUnauthorized, errors.New("user authentication required"))
		return false
	}
	return true
}

func projectIDFromParam(c *gin.Context, param string) (int, bool) {
	value := strings.TrimSpace(c.Param(param))
	if value == "" {
		AbortWithError(c, http.StatusBadRequest, fmt.Errorf("%s is required", param))
		return 0, false
	}

	if guid, err := objects.ParseGUID(value); err == nil {
		if guid.Type != ent.TypeProject || guid.ID <= 0 {
			AbortWithError(c, http.StatusBadRequest, errors.New("invalid project ID"))
			return 0, false
		}
		return guid.ID, true
	}

	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		AbortWithError(c, http.StatusBadRequest, errors.New("project ID must be a numeric id or AxonHub project GUID"))
		return 0, false
	}
	return id, true
}

func userCanAccessProjectWithScopes(user *ent.User, projectID int, requiredScopes ...scopes.ScopeSlug) bool {
	if user == nil {
		return false
	}
	if user.IsOwner {
		return true
	}

	membership := userProjectMembership(user, projectID)
	if membership == nil {
		return false
	}
	if membership.IsOwner {
		return true
	}

	for _, requiredScope := range requiredScopes {
		if !userHasProjectScope(user, membership, projectID, requiredScope) {
			return false
		}
	}
	return true
}

func userProjectMembership(user *ent.User, projectID int) *ent.UserProject {
	for _, membership := range user.Edges.ProjectUsers {
		if membership != nil && membership.ProjectID == projectID {
			return membership
		}
	}
	return nil
}

func userHasProjectScope(user *ent.User, membership *ent.UserProject, projectID int, requiredScope scopes.ScopeSlug) bool {
	required := string(requiredScope)
	if slices.Contains(membership.Scopes, required) {
		return true
	}

	for _, role := range user.Edges.Roles {
		if role != nil && role.ProjectID != nil && *role.ProjectID == projectID && slices.Contains(role.Scopes, required) {
			return true
		}
	}

	return false
}
