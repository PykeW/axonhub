package orchestrator

import (
	"context"
	"sort"

	"github.com/samber/lo"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
)

type RelayChannelPoolSelector struct {
	wrapped CandidateSelector
}

func WithRelayChannelPoolSelector(wrapped CandidateSelector) *RelayChannelPoolSelector {
	return &RelayChannelPoolSelector{wrapped: wrapped}
}

func (s *RelayChannelPoolSelector) Select(ctx context.Context, req *llm.Request) ([]*ChannelModelsCandidate, error) {
	candidates, err := s.wrapped.Select(ctx, req)
	if err != nil || len(candidates) == 0 {
		return candidates, err
	}

	value, ok := contexts.GetRelayAuthContext(ctx)
	if !ok || value == nil {
		return candidates, nil
	}
	relay, ok := value.(*biz.RelayAuthContext)
	if !ok || relay == nil || len(relay.ChannelPool.Channels) == 0 {
		return candidates, nil
	}
	if !relay.AllowsModel(req.Model) {
		return []*ChannelModelsCandidate{}, nil
	}

	priorityByChannelID := make(map[int]int, len(relay.ChannelPool.Channels))
	filtered := lo.Filter(candidates, func(candidate *ChannelModelsCandidate, _ int) bool {
		if candidate == nil || candidate.Channel == nil {
			return false
		}
		entry, ok := relay.PoolEntry(candidate.Channel.ID)
		if !ok || !entry.AllowsModel(req.Model) {
			return false
		}
		priorityByChannelID[candidate.Channel.ID] = entry.Priority
		if entry.Priority > 0 && (candidate.Priority == 0 || entry.Priority < candidate.Priority) {
			candidate.Priority = entry.Priority
		}
		return true
	})

	sort.SliceStable(filtered, func(i, j int) bool {
		left := priorityByChannelID[filtered[i].Channel.ID]
		right := priorityByChannelID[filtered[j].Channel.ID]
		if left == right {
			return false
		}
		return left < right
	})

	return filtered, nil
}
