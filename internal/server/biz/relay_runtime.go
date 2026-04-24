package biz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/looplj/axonhub/internal/ent"
)

var (
	ErrRelayAccessDenied        = errors.New("relay access denied")
	ErrRelayInsufficientBalance = errors.New("relay insufficient balance")
	ErrRelayQuotaExceeded       = errors.New("relay quota exceeded")
)

type RelayRuntimeService struct {
	resolver   RelayAuthResolver
	access     RelayAccessChecker
	settlement RelaySettlementRecorder
}

type RelayRuntimeOption func(*RelayRuntimeService)

func NewRelayRuntimeService() *RelayRuntimeService {
	return &RelayRuntimeService{}
}

func (s *RelayRuntimeService) SetResolver(resolver RelayAuthResolver) {
	if s != nil {
		s.resolver = resolver
	}
}

func (s *RelayRuntimeService) SetAccessChecker(access RelayAccessChecker) {
	if s != nil {
		s.access = access
	}
}

func (s *RelayRuntimeService) SetSettlementRecorder(settlement RelaySettlementRecorder) {
	if s != nil {
		s.settlement = settlement
	}
}

type RelayAuthResolver interface {
	ResolveRelayAuth(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error)
}

type RelayAccessChecker interface {
	CheckRelayAccess(ctx context.Context, relay *RelayAuthContext, input RelayAccessCheckInput) (*RelayAccessDecision, error)
}

type RelaySettlementRecorder interface {
	RecordRelayUsage(ctx context.Context, relay *RelayAuthContext, input RelayUsageSettlementInput) error
}

type RelayAuthContext struct {
	RelayKeyID   int
	APIKeyID     int
	ProjectID    int
	ProductID    int
	ProductCode  string
	ProductName  string
	ProviderType RelayProductProviderType
	Status       RelayKeyStatus
	BalanceMode  RelayKeyBalanceMode
	ExpiresAt    *time.Time
	Wallet       *RelayWalletSnapshot
	Quota        RelayKeyQuotaSnapshot
	DailyUsage   RelayUsageSnapshot
	ChannelPool  RelayChannelPool
	Metadata     map[string]any
}

type RelayKeyStatus string

const (
	RelayKeyStatusActive    RelayKeyStatus = "active"
	RelayKeyStatusSuspended RelayKeyStatus = "suspended"
	RelayKeyStatusExhausted RelayKeyStatus = "exhausted"
	RelayKeyStatusArchived  RelayKeyStatus = "archived"
)

type RelayKeyBalanceMode string

const (
	RelayKeyBalanceModePrepaid   RelayKeyBalanceMode = "prepaid"
	RelayKeyBalanceModeQuotaOnly RelayKeyBalanceMode = "quota_only"
)

type RelayWalletSnapshot struct {
	Currency        string
	AvailableAmount decimal.Decimal
	FrozenAmount    decimal.Decimal
	OverdraftLimit  decimal.Decimal
	Version         int64
}

type RelayKeyQuotaSnapshot struct {
	DailyRequestLimit *int64
	DailyTokenLimit   *int64
	MonthlyCostLimit  *decimal.Decimal
	ConcurrencyLimit  *int64
}

type RelayUsageSnapshot struct {
	RequestCount int64
	TotalTokens  int64
	TotalCharge  decimal.Decimal
}

type RelayChannelPool struct {
	AllowedModels []string
	Channels      []RelayChannelPoolEntry
}

type RelayChannelPoolEntry struct {
	ChannelID     int
	Priority      int
	Weight        int
	AllowFallback bool
	ModelFilter   []string
	MaxInflight   *int
}

type RelayAccessCheckInput struct {
	APIKey *ent.APIKey
	Now    time.Time
}

type RelayAccessDecision struct {
	Allowed    bool
	StatusCode int
	Code       string
	Message    string
}

type RelayUsageSettlementInput struct {
	UsageLog *ent.UsageLog
	Request  *ent.Request
}

func (s *RelayRuntimeService) ResolveAndCheckAccess(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, *RelayAccessDecision, error) {
	if s == nil || s.resolver == nil || apiKey == nil {
		return nil, nil, nil
	}

	relay, err := s.resolver.ResolveRelayAuth(ctx, apiKey)
	if err != nil || relay == nil {
		return relay, nil, err
	}

	if relay.APIKeyID == 0 {
		relay.APIKeyID = apiKey.ID
	}
	if relay.ProjectID == 0 {
		relay.ProjectID = apiKey.ProjectID
	}

	decision := s.defaultAccessDecision(relay, time.Now())
	if s.access != nil {
		decision, err = s.access.CheckRelayAccess(ctx, relay, RelayAccessCheckInput{
			APIKey: apiKey,
			Now:    time.Now(),
		})
		if err != nil {
			return relay, nil, err
		}
		if decision == nil {
			decision = allowRelayAccess()
		}
	}

	return relay, decision, nil
}

func (s *RelayRuntimeService) RecordUsageSettlement(ctx context.Context, relay *RelayAuthContext, input RelayUsageSettlementInput) error {
	if s == nil || s.settlement == nil || relay == nil || input.UsageLog == nil {
		return nil
	}
	return s.settlement.RecordRelayUsage(ctx, relay, input)
}

func (s *RelayRuntimeService) defaultAccessDecision(relay *RelayAuthContext, now time.Time) *RelayAccessDecision {
	if relay.Status != "" && relay.Status != RelayKeyStatusActive {
		return denyRelayAccess(http.StatusForbidden, "relay_key_inactive", fmt.Sprintf("relay key is %s", relay.Status))
	}
	if relay.ExpiresAt != nil && !relay.ExpiresAt.After(now) {
		return denyRelayAccess(http.StatusForbidden, "relay_key_expired", "relay key has expired")
	}
	if relay.BalanceMode == RelayKeyBalanceModePrepaid && relay.Wallet != nil {
		available := relay.Wallet.AvailableAmount.Add(relay.Wallet.OverdraftLimit)
		if available.LessThanOrEqual(decimal.Zero) {
			return denyRelayAccess(http.StatusPaymentRequired, "relay_balance_exhausted", "relay key balance exhausted")
		}
	}
	if relay.Quota.DailyRequestLimit != nil && relay.DailyUsage.RequestCount >= *relay.Quota.DailyRequestLimit {
		return denyRelayAccess(http.StatusForbidden, "relay_daily_request_quota_exceeded", "relay key daily request quota exceeded")
	}
	if relay.Quota.DailyTokenLimit != nil && relay.DailyUsage.TotalTokens >= *relay.Quota.DailyTokenLimit {
		return denyRelayAccess(http.StatusForbidden, "relay_daily_token_quota_exceeded", "relay key daily token quota exceeded")
	}
	return allowRelayAccess()
}

func allowRelayAccess() *RelayAccessDecision {
	return &RelayAccessDecision{Allowed: true, StatusCode: http.StatusOK}
}

func denyRelayAccess(statusCode int, code, message string) *RelayAccessDecision {
	return &RelayAccessDecision{Allowed: false, StatusCode: statusCode, Code: code, Message: message}
}

func (d *RelayAccessDecision) ErrorOrNil() error {
	if d == nil || d.Allowed {
		return nil
	}
	switch d.Code {
	case "relay_balance_exhausted":
		return fmt.Errorf("%w: %s", ErrRelayInsufficientBalance, d.Message)
	case "relay_daily_request_quota_exceeded", "relay_daily_token_quota_exceeded":
		return fmt.Errorf("%w: %s", ErrRelayQuotaExceeded, d.Message)
	default:
		return fmt.Errorf("%w: %s", ErrRelayAccessDenied, d.Message)
	}
}

func (c *RelayAuthContext) AllowedChannelIDs() []int {
	if c == nil || len(c.ChannelPool.Channels) == 0 {
		return nil
	}
	ids := make([]int, 0, len(c.ChannelPool.Channels))
	for _, entry := range c.ChannelPool.Channels {
		if entry.ChannelID > 0 {
			ids = append(ids, entry.ChannelID)
		}
	}
	return ids
}

func (c *RelayAuthContext) AllowsModel(model string) bool {
	if c == nil || model == "" || len(c.ChannelPool.AllowedModels) == 0 {
		return true
	}
	return matchRelayModelPatterns(c.ChannelPool.AllowedModels, model)
}

func (e RelayChannelPoolEntry) AllowsModel(model string) bool {
	if model == "" || len(e.ModelFilter) == 0 {
		return true
	}
	return matchRelayModelPatterns(e.ModelFilter, model)
}

func (c *RelayAuthContext) PoolEntry(channelID int) (RelayChannelPoolEntry, bool) {
	if c == nil {
		return RelayChannelPoolEntry{}, false
	}
	for _, entry := range c.ChannelPool.Channels {
		if entry.ChannelID == channelID {
			return entry, true
		}
	}
	return RelayChannelPoolEntry{}, false
}

func matchRelayModelPatterns(patterns []string, model string) bool {
	if len(patterns) == 0 {
		return true
	}
	if slices.Contains(patterns, model) {
		return true
	}
	for _, pattern := range patterns {
		if pattern == "*" || matchRelayWildcard(pattern, model) {
			return true
		}
	}
	return false
}

func matchRelayWildcard(pattern, value string) bool {
	if pattern == "" {
		return value == ""
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if parts[0] != "" && !strings.HasPrefix(value, parts[0]) {
		return false
	}
	if last := parts[len(parts)-1]; last != "" && !strings.HasSuffix(value, last) {
		return false
	}

	pos := len(parts[0])
	for _, part := range parts[1 : len(parts)-1] {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		pos += idx + len(part)
	}
	return true
}
