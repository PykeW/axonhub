package contexts

import "context"

func WithRelayAuthContext(ctx context.Context, relay any) context.Context {
	container := getContainer(ctx)
	container.RelayAuth = relay
	return withContainer(ctx, container)
}

func GetRelayAuthContext(ctx context.Context) (any, bool) {
	container := getContainer(ctx)
	return container.RelayAuth, container.RelayAuth != nil
}
