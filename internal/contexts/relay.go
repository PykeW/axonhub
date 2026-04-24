package contexts

import "context"

// WithRelayAuthContext stores relay runtime auth data without importing server/biz.
func WithRelayAuthContext(ctx context.Context, relayAuthContext any) context.Context {
	container := getContainer(ctx)
	container.RelayAuthContext = relayAuthContext

	return withContainer(ctx, container)
}

// GetRelayAuthContext retrieves relay runtime auth data for packages that know its concrete type.
func GetRelayAuthContext(ctx context.Context) (any, bool) {
	container := getContainer(ctx)
	return container.RelayAuthContext, container.RelayAuthContext != nil
}
