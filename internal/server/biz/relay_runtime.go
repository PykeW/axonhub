package biz

import (
	"context"

	"github.com/looplj/axonhub/internal/ent"
)

// RelayAuthContext carries request-scoped Relay/Sub-Key routing metadata.
// Keep this type in biz; contexts stores it as any to avoid an import cycle.
type RelayAuthContext struct {
	APIKeyID          int
	ProjectID         int
	ProductID         int
	ProductCode       string
	ProviderType      RelayProductProviderType
	AllowedChannelIDs []int
	AllowedModelIDs   []string
	SharedCapacity    bool
}

func (c *RelayAuthContext) HasRoutingConstraints() bool {
	return c != nil && len(c.AllowedChannelIDs) > 0
}

// AuthenticateRelayAPIKey is the post-API-key-auth Relay hook.
// It is intentionally a no-op until relay_keys code generation lands; callers can
// already consume a RelayAuthContext without coupling contexts to biz.
func (s *AuthService) AuthenticateRelayAPIKey(_ context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
	if apiKey == nil {
		return nil, nil
	}

	return nil, nil
}
