package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/apikey"
	"github.com/looplj/axonhub/internal/ent/request"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
)

// WithAPIKeyAuth 中间件用于验证 API key.
func WithAPIKeyAuth(auth *biz.AuthService) gin.HandlerFunc {
	return WithAPIKeyConfig(auth, nil)
}

func WithAPIKeyConfig(auth *biz.AuthService, config *APIKeyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, err := ExtractAPIKeyFromRequest(c.Request, config)
		// DO NOT ALLOW USE NO AUTH API KEY DIRECTLY.
		if key == biz.NoAuthAPIKeyValue {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid API key"))
			return
		}

		var apiKey *ent.APIKey
		if err == nil {
			apiKey, err = auth.AuthenticateAPIKey(c.Request.Context(), key)
		}
		if err != nil {
			apiKey, err = auth.AuthenticateNoAuth(c.Request.Context())
		}
		if err != nil {
			if ent.IsNotFound(err) || errors.Is(err, biz.ErrInvalidAPIKey) {
				AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid API key"))
			} else {
				log.Error(c.Request.Context(), "Failed to validate API key", log.Cause(err))
				AbortWithError(c, http.StatusInternalServerError, errors.New("Failed to validate API key"))
			}

			return
		}

		ctx := contexts.WithAPIKey(c.Request.Context(), apiKey)

		if apiKey.Edges.Project != nil {
			ctx = contexts.WithProjectID(ctx, apiKey.Edges.Project.ID)
		}

		ctx, err = attachRelayAuthContext(ctx, auth, apiKey)
		if err != nil {
			abortRelayAuthError(c, err)
			return
		}
		defer releaseRelayAuthContext(ctx)

		ctx, err = withAPIKeyPrincipal(ctx, apiKey)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid authentication context"))
			return
		}

		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func WithJWTAuth(auth *biz.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := ExtractAPIKeyFromRequest(c.Request, &APIKeyConfig{
			Headers:       []string{"Authorization"},
			RequireBearer: true,
		})
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, err)
			return
		}

		user, err := auth.AuthenticateJWTToken(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, biz.ErrInvalidJWT) {
				AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid token"))
			} else {
				AbortWithError(c, http.StatusInternalServerError, errors.New("Failed to validate token"))
			}

			return
		}

		ctx := contexts.WithUser(c.Request.Context(), user)

		ctx, err = withUserPrincipal(ctx, user)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid authentication context"))
			return
		}

		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

var apiKeyAuthConfig = &APIKeyConfig{
	Headers:       []string{"Authorization"},
	RequireBearer: true,
}

// WithOpenAPIAuth allows API key auth for createLLMAPIKey only.
func WithOpenAPIAuth(auth *biz.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, err := ExtractAPIKeyFromRequest(c.Request, apiKeyAuthConfig)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, err)
			return
		}

		apiKey, err := auth.AuthenticateAPIKey(c.Request.Context(), key)
		if err != nil {
			if ent.IsNotFound(err) || errors.Is(err, biz.ErrInvalidAPIKey) {
				AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid API key"))
			} else {
				AbortWithError(c, http.StatusInternalServerError, errors.New("Failed to validate API key"))
			}

			return
		}

		if apiKey.Type != apikey.TypeServiceAccount {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid API key"))
			return
		}

		ctx := contexts.WithAPIKey(c.Request.Context(), apiKey)
		if apiKey.Edges.Project != nil {
			ctx = contexts.WithProjectID(ctx, apiKey.Edges.Project.ID)
		}

		ctx, err = withAPIKeyPrincipal(ctx, apiKey)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid authentication context"))
			return
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// WithGeminiKeyAuth be compatible with Gemini query key authentication.
// https://ai.google.dev/api/generate-content?hl=zh-cn#text_gen_text_only_prompt-SHELL
func WithGeminiKeyAuth(auth *biz.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Query("key")
		if key == "" {
			var err error

			key, err = ExtractAPIKeyFromRequest(c.Request, nil)
			if err != nil {
				AbortWithError(c, http.StatusUnauthorized, err)
				return
			}
		}

		apiKey, err := auth.AuthenticateAPIKey(c.Request.Context(), key)
		if err != nil {
			if ent.IsNotFound(err) || errors.Is(err, biz.ErrInvalidAPIKey) {
				AbortWithError(c, http.StatusUnauthorized, biz.ErrInvalidAPIKey)
			} else {
				AbortWithError(c, http.StatusInternalServerError, errors.New("Failed to validate API key"))
			}

			return
		}

		// 将 API key entity 保存到 context 中
		ctx := contexts.WithAPIKey(c.Request.Context(), apiKey)

		if apiKey.Edges.Project != nil {
			ctx = contexts.WithProjectID(ctx, apiKey.Edges.Project.ID)
		}

		ctx, err = attachRelayAuthContext(ctx, auth, apiKey)
		if err != nil {
			abortRelayAuthError(c, err)
			return
		}
		defer releaseRelayAuthContext(ctx)

		ctx, err = withAPIKeyPrincipal(ctx, apiKey)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, errors.New("Invalid authentication context"))
			return
		}

		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// WithSource 中间件用于设置请求来源到 context 中.
func WithSource(source request.Source) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := contexts.WithSource(c.Request.Context(), source)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func withUserPrincipal(ctx context.Context, user *ent.User) (context.Context, error) {
	principal := authz.Principal{Type: authz.PrincipalTypeUser, UserID: &user.ID}
	return authz.WithPrincipal(ctx, principal)
}

func attachRelayAuthContext(ctx context.Context, auth *biz.AuthService, apiKey *ent.APIKey) (context.Context, error) {
	relayAuthContext, err := auth.AuthenticateRelayAPIKey(ctx, apiKey)
	if err != nil {
		return ctx, err
	}
	if relayAuthContext != nil {
		ctx = contexts.WithRelayAuthContext(ctx, relayAuthContext)
	}

	return ctx, nil
}

func releaseRelayAuthContext(ctx context.Context) {
	value, ok := contexts.GetRelayAuthContext(ctx)
	if !ok || value == nil {
		return
	}
	relayAuthContext, ok := value.(*biz.RelayAuthContext)
	if ok && relayAuthContext != nil {
		relayAuthContext.ReleaseInflight()
	}
}

func abortRelayAuthError(c *gin.Context, err error) {
	var relayErr *biz.RelayAuthError
	if errors.As(err, &relayErr) {
		_ = c.Error(relayErr)
		c.AbortWithStatusJSON(relayErr.HTTPStatus(), objects.ErrorResponse{
			Error: objects.Error{
				Type:    relayErr.ErrorCode(),
				Message: relayErr.ErrorMessage(),
			},
		})
		return
	}

	log.Error(c.Request.Context(), "Failed to validate relay authentication context", log.Cause(err))
	AbortWithError(c, http.StatusInternalServerError, errors.New("Failed to validate relay authentication context"))
}

func withAPIKeyPrincipal(ctx context.Context, key *ent.APIKey) (context.Context, error) {
	principal := authz.Principal{Type: authz.PrincipalTypeAPIKey, APIKeyID: &key.ID}
	if key.Edges.Project != nil {
		projectID := key.Edges.Project.ID
		principal.ProjectID = &projectID
	}

	return authz.WithPrincipal(ctx, principal)
}
