package orchestrator

import "github.com/looplj/axonhub/internal/server/biz"

func asRelayAuthContext(value any) *biz.RelayAuthContext {
	relayAuthContext, _ := value.(*biz.RelayAuthContext)
	return relayAuthContext
}
