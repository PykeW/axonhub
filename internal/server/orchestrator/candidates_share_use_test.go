package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
)

type shareUseMockSelector struct {
	candidates []*ChannelModelsCandidate
}

func (m *shareUseMockSelector) Select(context.Context, *llm.Request) ([]*ChannelModelsCandidate, error) {
	return m.candidates, nil
}

func TestShareUseStrategySelector_Select_OwnFirstWithLegacyFallback(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	selector := WithShareUseStrategySelector(&shareUseMockSelector{candidates: []*ChannelModelsCandidate{
		newShareUseTestCandidate("own-late", 7, objects.ChannelVisibilityPrivate, now.Add(2*time.Hour)),
		newShareUseTestCandidate("shared-early", 8, objects.ChannelVisibilityShared, now.Add(30*time.Minute)),
		newShareUseTestCandidate("own-early", 7, objects.ChannelVisibilityShared, now.Add(15*time.Minute)),
		newShareUseTestCandidate("other-private", 9, objects.ChannelVisibilityPrivate, now.Add(10*time.Minute)),
		newLegacyShareUseTestCandidate("legacy"),
	}}, &ent.APIKey{
		UserID: 7,
		Profiles: &objects.APIKeyProfiles{
			ActiveProfile: "default",
			Profiles: []objects.APIKeyProfile{{
				Name:        "default",
				UseStrategy: objects.APIKeyUseStrategyPreferOwn,
			}},
		},
	})

	got, err := selector.Select(context.Background(), &llm.Request{Model: "gpt-4"})
	require.NoError(t, err)
	require.Equal(t, []string{"own-early", "own-late", "shared-early", "legacy"}, shareUseCandidateNames(got))
	require.Equal(t, 0, got[0].Priority)
	require.Equal(t, shareUseBucketPriorityOffset, got[2].Priority)
	require.Equal(t, 2*shareUseBucketPriorityOffset, got[3].Priority)
}

func TestShareUseStrategySelector_Select_SharedFirst(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	selector := WithShareUseStrategySelector(&shareUseMockSelector{candidates: []*ChannelModelsCandidate{
		newShareUseTestCandidate("own", 7, objects.ChannelVisibilityPrivate, now.Add(2*time.Hour)),
		newShareUseTestCandidate("shared-late", 8, objects.ChannelVisibilityShared, now.Add(45*time.Minute)),
		newShareUseTestCandidate("shared-early", 9, objects.ChannelVisibilityShared, now.Add(15*time.Minute)),
		newLegacyShareUseTestCandidate("legacy"),
	}}, &ent.APIKey{
		UserID: 7,
		Profiles: &objects.APIKeyProfiles{
			ActiveProfile: "default",
			Profiles: []objects.APIKeyProfile{{
				Name:        "default",
				UseStrategy: objects.APIKeyUseStrategySharedFirst,
			}},
		},
	})

	got, err := selector.Select(context.Background(), &llm.Request{Model: "gpt-4"})
	require.NoError(t, err)
	require.Equal(t, []string{"shared-early", "shared-late", "own", "legacy"}, shareUseCandidateNames(got))
	require.Equal(t, 0, got[0].Priority)
	require.Equal(t, shareUseBucketPriorityOffset, got[2].Priority)
}

func TestShareUseStrategySelector_Select_SharedOnly(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	selector := WithShareUseStrategySelector(&shareUseMockSelector{candidates: []*ChannelModelsCandidate{
		newShareUseTestCandidate("own", 7, objects.ChannelVisibilityPrivate, now.Add(time.Hour)),
		newShareUseTestCandidate("shared", 8, objects.ChannelVisibilityShared, now.Add(30*time.Minute)),
		newLegacyShareUseTestCandidate("legacy"),
	}}, &ent.APIKey{
		UserID: 7,
		Profiles: &objects.APIKeyProfiles{
			ActiveProfile: "default",
			Profiles: []objects.APIKeyProfile{{
				Name:        "default",
				UseStrategy: objects.APIKeyUseStrategySharedOnly,
			}},
		},
	})

	got, err := selector.Select(context.Background(), &llm.Request{Model: "gpt-4"})
	require.NoError(t, err)
	require.Equal(t, []string{"shared"}, shareUseCandidateNames(got))
}

func TestShareUseStrategySelector_Select_UsesContextUserFallback(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	selector := WithShareUseStrategySelector(&shareUseMockSelector{candidates: []*ChannelModelsCandidate{
		newShareUseTestCandidate("shared", 8, objects.ChannelVisibilityShared, now.Add(30*time.Minute)),
		newShareUseTestCandidate("own", 7, objects.ChannelVisibilityPrivate, now.Add(15*time.Minute)),
	}}, nil)

	ctx := contexts.WithUser(context.Background(), &ent.User{ID: 7})
	got, err := selector.Select(ctx, &llm.Request{Model: "gpt-4"})
	require.NoError(t, err)
	require.Equal(t, []string{"own", "shared"}, shareUseCandidateNames(got))
}

func newShareUseTestCandidate(name string, ownerUserID int, visibility objects.ChannelVisibility, nextRefreshAt time.Time) *ChannelModelsCandidate {
	return &ChannelModelsCandidate{
		Channel: &biz.Channel{
			Channel: &ent.Channel{
				Name: name,
				Settings: &objects.ChannelSettings{
					Share: &objects.ChannelShareSettings{
						OwnerUserID:   &objects.GUID{Type: ent.TypeUser, ID: ownerUserID},
						Visibility:    visibility,
						NextRefreshAt: &nextRefreshAt,
					},
				},
			},
		},
		Models: []biz.ChannelModelEntry{{RequestModel: "gpt-4", ActualModel: "gpt-4"}},
	}
}

func newLegacyShareUseTestCandidate(name string) *ChannelModelsCandidate {
	return &ChannelModelsCandidate{
		Channel: &biz.Channel{
			Channel: &ent.Channel{Name: name},
		},
		Models: []biz.ChannelModelEntry{{RequestModel: "gpt-4", ActualModel: "gpt-4"}},
	}
}

func shareUseCandidateNames(candidates []*ChannelModelsCandidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Channel.Name)
	}

	return names
}
