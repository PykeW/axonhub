package biz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
)

type RelayAdminServiceParams struct {
	fx.In

	Ent            *ent.Client
	ProductService *RelayProductService
	APIKeyService  *APIKeyService
}

type RelayAdminService struct {
	*AbstractService

	productService *RelayProductService
	apiKeyService  *APIKeyService
}

type RelayHealthStatus string

type RelayDerivedState string

type RelayWalletLedgerType string

type RelayFailureStage string

type RelaySettlementStatus string

const (
	RelayHealthStatusHealthy     RelayHealthStatus = "healthy"
	RelayHealthStatusDegraded    RelayHealthStatus = "degraded"
	RelayHealthStatusUnavailable RelayHealthStatus = "unavailable"

	RelayDerivedStateExpired              RelayDerivedState = "expired"
	RelayDerivedStateLowBalance           RelayDerivedState = "low_balance"
	RelayDerivedStateQuotaReached         RelayDerivedState = "quota_reached"
	RelayDerivedStateUpstreamPoolDegraded RelayDerivedState = "upstream_pool_degraded"

	RelayWalletLedgerTypeRecharge   RelayWalletLedgerType = "recharge"
	RelayWalletLedgerTypeCharge     RelayWalletLedgerType = "charge"
	RelayWalletLedgerTypeRefund     RelayWalletLedgerType = "refund"
	RelayWalletLedgerTypeAdjustment RelayWalletLedgerType = "adjustment"
	RelayWalletLedgerTypeFreeze     RelayWalletLedgerType = "freeze"
	RelayWalletLedgerTypeUnfreeze   RelayWalletLedgerType = "unfreeze"

	RelayFailureStageAuth          RelayFailureStage = "auth"
	RelayFailureStageKeyValidation RelayFailureStage = "key_validation"
	RelayFailureStageRouting       RelayFailureStage = "routing"
	RelayFailureStageUpstream      RelayFailureStage = "upstream"
	RelayFailureStageSettlement    RelayFailureStage = "settlement"
	RelayFailureStageNone          RelayFailureStage = "none"

	RelaySettlementStatusCharged RelaySettlementStatus = "charged"
	RelaySettlementStatusDelayed RelaySettlementStatus = "delayed"
	RelaySettlementStatusSkipped RelaySettlementStatus = "skipped"
)

type RelayAdminProductChannel struct {
	ID                    string                    `json:"id"`
	ChannelID             string                    `json:"channelId"`
	ChannelName           string                    `json:"channelName"`
	Provider              RelayProductProviderType  `json:"provider"`
	Priority              int                       `json:"priority"`
	Weight                int                       `json:"weight"`
	Status                RelayProductChannelStatus `json:"status"`
	Health                RelayHealthStatus         `json:"health"`
	AllowFallback         bool                      `json:"allowFallback"`
	ModelFilter           []string                  `json:"modelFilter"`
	QuotaRemainingPercent int                       `json:"quotaRemainingPercent"`
	ErrorRatePercent      float64                   `json:"errorRatePercent"`
	LatencyMs             int                       `json:"latencyMs"`
	UnavailableReason     string                    `json:"unavailableReason,omitempty"`
	LastCheckedAt         string                    `json:"lastCheckedAt"`
}

type RelayAdminProduct struct {
	ID                    string                     `json:"id"`
	Code                  string                     `json:"code"`
	Name                  string                     `json:"name"`
	ProviderType          RelayProductProviderType   `json:"providerType"`
	Status                RelayProductStatus         `json:"status"`
	BillingMode           RelayProductBillingMode    `json:"billingMode"`
	Description           string                     `json:"description"`
	AllowedModels         []string                   `json:"allowedModels"`
	DefaultTimeoutMs      int                        `json:"defaultTimeoutMs"`
	PoolHealth            RelayHealthStatus          `json:"poolHealth"`
	ChannelPool           []RelayAdminProductChannel `json:"channelPool"`
	KeyCount              int                        `json:"keyCount"`
	ActiveKeyCount        int                        `json:"activeKeyCount"`
	MonthlyRequestCount   int64                      `json:"monthlyRequestCount"`
	MonthlyTokenCount     int64                      `json:"monthlyTokenCount"`
	MonthlyCost           float64                    `json:"monthlyCost"`
	CreatedAt             string                     `json:"createdAt"`
	UpdatedAt             string                     `json:"updatedAt"`
	AccessMode            RelayProductAccessMode     `json:"accessMode,omitempty"`
	Currency              string                     `json:"currency,omitempty"`
	RequestTimeoutSeconds int                        `json:"requestTimeoutSeconds,omitempty"`
	ListPriceConfig       map[string]any             `json:"listPriceConfig,omitempty"`
}

type RelayKeyLimitSnapshot struct {
	DailyRequestLimit int64   `json:"dailyRequestLimit"`
	DailyTokenLimit   int64   `json:"dailyTokenLimit"`
	MonthlyCostLimit  float64 `json:"monthlyCostLimit"`
	ConcurrencyLimit  int64   `json:"concurrencyLimit"`
}

type RelayKeyUsageSnapshot struct {
	TodayRequests int64   `json:"todayRequests"`
	TodayTokens   int64   `json:"todayTokens"`
	MonthlyCost   float64 `json:"monthlyCost"`
	LastFailureAt string  `json:"lastFailureAt,omitempty"`
	RecentFailure string  `json:"recentFailure,omitempty"`
}

type RelayAdminKey struct {
	ID            string                `json:"id"`
	APIKeyID      string                `json:"apiKeyId"`
	ProjectID     string                `json:"projectId"`
	ProjectName   string                `json:"projectName"`
	ProductID     string                `json:"productId"`
	ProductName   string                `json:"productName"`
	Name          string                `json:"name"`
	MaskedKey     string                `json:"maskedKey"`
	Status        RelayKeyStatus        `json:"status"`
	DerivedStates []RelayDerivedState   `json:"derivedStates"`
	BalanceMode   RelayKeyBalanceMode   `json:"balanceMode"`
	ExpiresAt     string                `json:"expiresAt,omitempty"`
	CreatedAt     string                `json:"createdAt"`
	LastUsedAt    string                `json:"lastUsedAt,omitempty"`
	BaseURL       string                `json:"baseUrl"`
	Limits        RelayKeyLimitSnapshot `json:"limits"`
	Usage         RelayKeyUsageSnapshot `json:"usage"`
}

type RelayAdminWallet struct {
	ID                  string  `json:"id"`
	RelayKeyID          string  `json:"relayKeyId"`
	Currency            string  `json:"currency"`
	AvailableAmount     float64 `json:"availableAmount"`
	FrozenAmount        float64 `json:"frozenAmount"`
	TotalRecharged      float64 `json:"totalRecharged"`
	TotalSpent          float64 `json:"totalSpent"`
	CreditLimit         float64 `json:"creditLimit"`
	LowBalanceThreshold float64 `json:"lowBalanceThreshold"`
	UpdatedAt           string  `json:"updatedAt"`
}

type RelayAdminWalletLedgerEntry struct {
	ID           string                `json:"id"`
	RelayKeyID   string                `json:"relayKeyId"`
	Type         RelayWalletLedgerType `json:"type"`
	Amount       float64               `json:"amount"`
	Currency     string                `json:"currency"`
	BalanceAfter float64               `json:"balanceAfter"`
	ReferenceID  string                `json:"referenceId,omitempty"`
	Operator     string                `json:"operator"`
	Note         string                `json:"note"`
	CreatedAt    string                `json:"createdAt"`
}

type RelayDailyUsageSummaryView struct {
	RelayKeyID       string  `json:"relayKeyId"`
	StatDate         string  `json:"statDate"`
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalCost        float64 `json:"totalCost"`
}

type RelayRequestTraceView struct {
	ID                 string                   `json:"id"`
	RequestID          string                   `json:"requestId"`
	CreatedAt          string                   `json:"createdAt"`
	ProjectName        string                   `json:"projectName"`
	KeyName            string                   `json:"keyName"`
	ProductName        string                   `json:"productName"`
	ModelID            string                   `json:"modelId"`
	ChannelName        string                   `json:"channelName,omitempty"`
	Provider           RelayProductProviderType `json:"provider,omitempty"`
	Status             string                   `json:"status"`
	FailureStage       RelayFailureStage        `json:"failureStage"`
	LatencyMs          int64                    `json:"latencyMs,omitempty"`
	ResponseStatusCode int                      `json:"responseStatusCode,omitempty"`
	Charged            bool                     `json:"charged"`
	ChargeAmount       float64                  `json:"chargeAmount"`
	SettlementStatus   RelaySettlementStatus    `json:"settlementStatus"`
	UsageLogID         string                   `json:"usageLogId,omitempty"`
	LedgerEntryID      string                   `json:"ledgerEntryId,omitempty"`
	PromptTokens       int64                    `json:"promptTokens"`
	CompletionTokens   int64                    `json:"completionTokens"`
	ErrorMessage       string                   `json:"errorMessage,omitempty"`
}

type RelayChannelPoolHealthView struct {
	ProductID           string                     `json:"productId"`
	ProductName         string                     `json:"productName"`
	Status              RelayHealthStatus          `json:"status"`
	HealthyChannels     int                        `json:"healthyChannels"`
	DegradedChannels    int                        `json:"degradedChannels"`
	UnavailableChannels int                        `json:"unavailableChannels"`
	RiskReason          string                     `json:"riskReason,omitempty"`
	Channels            []RelayAdminProductChannel `json:"channels"`
}

type ProjectRelayOverviewView struct {
	Products       []RelayAdminProduct          `json:"products"`
	Keys           []RelayAdminKey              `json:"keys"`
	Wallets        []RelayAdminWallet           `json:"wallets"`
	RecentRequests []RelayRequestTraceView      `json:"recentRequests"`
	Usage          []RelayDailyUsageSummaryView `json:"usage"`
}

type ProjectRelayUsageView struct {
	Wallets        []RelayAdminWallet            `json:"wallets"`
	LedgerEntries  []RelayAdminWalletLedgerEntry `json:"ledgerEntries"`
	Usage          []RelayDailyUsageSummaryView  `json:"usage"`
	RecentRequests []RelayRequestTraceView       `json:"recentRequests"`
}

type RelayAdminProductCreateInput struct {
	Code             string                   `json:"code"`
	Name             string                   `json:"name"`
	ProviderType     RelayProductProviderType `json:"providerType"`
	BillingMode      RelayProductBillingMode  `json:"billingMode"`
	AllowedModels    []string                 `json:"allowedModels"`
	DefaultTimeoutMs int                      `json:"defaultTimeoutMs"`
	Description      string                   `json:"description"`
	Status           RelayProductStatus       `json:"status"`
}

type RelayAdminProductUpdateInput struct {
	Name             *string                  `json:"name,omitempty"`
	BillingMode      *RelayProductBillingMode `json:"billingMode,omitempty"`
	AllowedModels    []string                 `json:"allowedModels,omitempty"`
	DefaultTimeoutMs *int                     `json:"defaultTimeoutMs,omitempty"`
	Description      *string                  `json:"description,omitempty"`
	Status           *RelayProductStatus      `json:"status,omitempty"`
}

type RelayAdminChannelBindingInput struct {
	ProductID     string   `json:"productId"`
	ChannelID     string   `json:"channelId"`
	Priority      int      `json:"priority"`
	Weight        int      `json:"weight"`
	ModelFilter   []string `json:"modelFilter"`
	AllowFallback *bool    `json:"allowFallback,omitempty"`
}

type RelayAdminCreateKeyInput struct {
	ProjectID      string                `json:"projectId"`
	ProductID      string                `json:"productId"`
	Name           string                `json:"name"`
	ExpiresAt      string                `json:"expiresAt"`
	BalanceMode    RelayKeyBalanceMode   `json:"balanceMode"`
	InitialBalance float64               `json:"initialBalance"`
	Limits         RelayKeyLimitSnapshot `json:"limits"`
}

type RelayAdminKeyStatusInput struct {
	Status RelayKeyStatus `json:"status"`
	Note   string         `json:"note"`
}

type RelayAdminKeyLimitsInput struct {
	Limits            *RelayKeyLimitSnapshot `json:"limits,omitempty"`
	DailyRequestLimit *int64                 `json:"dailyRequestLimit,omitempty"`
	DailyTokenLimit   *int64                 `json:"dailyTokenLimit,omitempty"`
	MonthlyCostLimit  *float64               `json:"monthlyCostLimit,omitempty"`
	ConcurrencyLimit  *int64                 `json:"concurrencyLimit,omitempty"`
}

type RelayAdminRechargeInput struct {
	RelayKeyID string  `json:"relayKeyId"`
	Amount     float64 `json:"amount"`
	Note       string  `json:"note"`
}

func NewRelayAdminService(params RelayAdminServiceParams) *RelayAdminService {
	return &RelayAdminService{
		AbstractService: &AbstractService{db: params.Ent},
		productService:  params.ProductService,
		apiKeyService:   params.APIKeyService,
	}
}

func (s *RelayAdminService) ListProducts(ctx context.Context) ([]RelayAdminProduct, error) {
	return s.listProductRows(ctx)
}

func (s *RelayAdminService) GetProduct(ctx context.Context, id int) (*RelayAdminProduct, error) {
	product, err := s.loadProductRow(ctx, id)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (s *RelayAdminService) CreateProduct(ctx context.Context, input RelayAdminProductCreateInput) (*RelayAdminProduct, error) {
	if s.productService == nil {
		return nil, fmt.Errorf("relay product service is not configured")
	}
	priceConfig := map[string]any{}
	if strings.TrimSpace(input.Description) != "" {
		priceConfig["description"] = strings.TrimSpace(input.Description)
	}
	product, err := s.productService.CreateRelayProduct(ctx, RelayProductCreateInput{
		Code:                  input.Code,
		Name:                  input.Name,
		ProviderType:          input.ProviderType,
		BillingMode:           input.BillingMode,
		Status:                input.Status,
		AllowedModels:         input.AllowedModels,
		RequestTimeoutSeconds: relayTimeoutSeconds(input.DefaultTimeoutMs),
		ListPriceConfig:       priceConfig,
	})
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, product.ID)
}

func (s *RelayAdminService) UpdateProduct(ctx context.Context, id int, input RelayAdminProductUpdateInput) (*RelayAdminProduct, error) {
	if s.productService == nil {
		return nil, fmt.Errorf("relay product service is not configured")
	}
	existing, err := s.productService.getRelayProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	priceConfig := existing.ListPriceConfig
	if priceConfig == nil {
		priceConfig = map[string]any{}
	}
	if input.Description != nil {
		priceConfig["description"] = strings.TrimSpace(*input.Description)
	}
	update := RelayProductUpdateInput{
		Name:            input.Name,
		BillingMode:     input.BillingMode,
		Status:          input.Status,
		AllowedModels:   input.AllowedModels,
		ListPriceConfig: priceConfig,
	}
	if input.DefaultTimeoutMs != nil {
		seconds := relayTimeoutSeconds(*input.DefaultTimeoutMs)
		update.RequestTimeoutSeconds = &seconds
	}
	if input.Description == nil {
		update.ListPriceConfig = nil
	}
	if _, err := s.productService.UpdateRelayProduct(ctx, id, update); err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, id)
}

func (s *RelayAdminService) CreateProductChannelBinding(ctx context.Context, input RelayAdminChannelBindingInput) (*RelayAdminProductChannel, error) {
	if s.productService == nil {
		return nil, fmt.Errorf("relay product service is not configured")
	}
	productID, err := parseRelayAdminID(input.ProductID, "productId")
	if err != nil {
		return nil, err
	}
	channelID, err := parseRelayAdminID(input.ChannelID, "channelId")
	if err != nil {
		return nil, err
	}
	allowFallback := true
	if input.AllowFallback != nil {
		allowFallback = *input.AllowFallback
	}
	modelFilter := map[string]any{}
	if len(input.ModelFilter) > 0 {
		modelFilter["models"] = input.ModelFilter
	}
	binding, err := s.productService.CreateRelayProductChannelBinding(ctx, RelayProductChannelBindingInput{
		ProductID:     productID,
		ChannelID:     channelID,
		Priority:      input.Priority,
		Weight:        input.Weight,
		Status:        RelayProductChannelStatusActive,
		AllowFallback: &allowFallback,
		ModelFilter:   modelFilter,
	})
	if err != nil {
		return nil, err
	}
	product, err := s.GetProduct(ctx, binding.ProductID)
	if err != nil {
		return nil, err
	}
	for _, ch := range product.ChannelPool {
		if ch.ID == relayStringID(binding.ID) {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("relay product channel binding %d not found", binding.ID)
}

func (s *RelayAdminService) ListKeys(ctx context.Context, projectID *int) ([]RelayAdminKey, error) {
	return s.listKeys(ctx, projectID, 0)
}

func (s *RelayAdminService) GetKey(ctx context.Context, id int) (*RelayAdminKey, error) {
	keys, err := s.listKeys(ctx, nil, id)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("relay key %d not found", id)
	}
	return &keys[0], nil
}

func (s *RelayAdminService) CreateKey(ctx context.Context, input RelayAdminCreateKeyInput) (*RelayAdminKey, error) {
	projectID, err := s.resolveProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	productID, err := parseRelayAdminID(input.ProductID, "productId")
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("relay key name is required")
	}
	balanceMode := input.BalanceMode
	if balanceMode == "" {
		balanceMode = RelayKeyBalanceModePrepaid
	}
	if balanceMode != RelayKeyBalanceModePrepaid && balanceMode != RelayKeyBalanceModeQuotaOnly {
		return nil, fmt.Errorf("unsupported relay key balance mode %q", balanceMode)
	}
	var expiresAt any
	if strings.TrimSpace(input.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.ExpiresAt))
		if err != nil {
			return nil, fmt.Errorf("invalid expiresAt: %w", err)
		}
		expiresAt = parsed
	}
	if input.InitialBalance < 0 {
		return nil, fmt.Errorf("initialBalance cannot be negative")
	}
	apiKeyValue, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	userID := nullableInt(0)
	if user, ok := contexts.GetUser(ctx); ok && user != nil && user.ID > 0 {
		userID = user.ID
	}
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin relay key transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	scopesJSON := `["read_channels","write_requests"]`
	profilesJSON := `{}`
	apiKeyID, err := execRelayInsertTx(ctx, tx, dialectName,
		fmt.Sprintf("INSERT INTO api_keys (key, name, type, status, scopes, profiles, project_id, user_id) VALUES (%s)", strings.Join(relayPlaceholders(dialectName, 8, 1), ",")),
		apiKeyValue, name, "user", "enabled", scopesJSON, profilesJSON, projectID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backing api key: %w", err)
	}
	relayKeyID, err := execRelayInsertTx(ctx, tx, dialectName,
		fmt.Sprintf(`INSERT INTO relay_keys (api_key_id, project_id, product_id, display_name, status, balance_mode, daily_request_limit, daily_token_limit, monthly_cost_limit, concurrency_limit, expires_at)
VALUES (%s)`, strings.Join(relayPlaceholders(dialectName, 11, 1), ",")),
		apiKeyID, projectID, productID, name, string(RelayKeyStatusActive), string(balanceMode), nullablePositiveInt64(input.Limits.DailyRequestLimit), nullablePositiveInt64(input.Limits.DailyTokenLimit), nullablePositiveDecimal(input.Limits.MonthlyCostLimit), nullablePositiveInt64(input.Limits.ConcurrencyLimit), expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay key: %w", err)
	}
	initialBalance := decimal.NewFromFloat(input.InitialBalance)
	_, err = execRelayInsertTx(ctx, tx, dialectName,
		fmt.Sprintf("INSERT INTO relay_wallets (relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version) VALUES (%s)", strings.Join(relayPlaceholders(dialectName, 7, 1), ",")),
		relayKeyID, projectID, RelayProductDefaultCurrency, initialBalance.String(), "0", "0", 1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay wallet: %w", err)
	}
	if initialBalance.GreaterThan(decimal.Zero) {
		_, err = insertRelayAdminLedgerTx(ctx, tx, dialectName, relayAdminLedgerInsert{
			RelayKeyID:     relayKeyID,
			ProjectID:      projectID,
			Direction:      "credit",
			Scene:          "recharge",
			Amount:         initialBalance,
			BalanceBefore:  decimal.Zero,
			BalanceAfter:   initialBalance,
			IdempotencyKey: fmt.Sprintf("initial_recharge:%d", relayKeyID),
			Remark:         "initial balance",
		})
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit relay key transaction: %w", err)
	}
	committed = true
	if s.apiKeyService != nil {
		s.apiKeyService.invalidateAPIKeyCaches(ctx, apiKeyValue)
	}
	return s.GetKey(ctx, relayKeyID)
}

func (s *RelayAdminService) UpdateKeyStatus(ctx context.Context, id int, input RelayAdminKeyStatusInput) (*RelayAdminKey, error) {
	if input.Status != RelayKeyStatusActive && input.Status != RelayKeyStatusSuspended && input.Status != RelayKeyStatusExhausted && input.Status != RelayKeyStatusArchived {
		return nil, fmt.Errorf("unsupported relay key status %q", input.Status)
	}
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	apiKeyID, apiKeyValue, err := s.relayKeyAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin update key status transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := fmt.Sprintf("UPDATE relay_keys SET status = %s, updated_at = CURRENT_TIMESTAMP WHERE id = %s AND deleted_at = 0", relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2))
	result, err := tx.ExecContext(ctx, query, string(input.Status), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update relay key status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, fmt.Errorf("relay key %d not found", id)
	}
	apiStatus := "disabled"
	if input.Status == RelayKeyStatusActive {
		apiStatus = "enabled"
	} else if input.Status == RelayKeyStatusArchived {
		apiStatus = "archived"
	}
	apiQuery := fmt.Sprintf("UPDATE api_keys SET status = %s, updated_at = CURRENT_TIMESTAMP WHERE id = %s", relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2))
	if _, err := tx.ExecContext(ctx, apiQuery, apiStatus, apiKeyID); err != nil {
		return nil, fmt.Errorf("failed to update backing api key status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit update key status transaction: %w", err)
	}
	committed = true

	if s.apiKeyService != nil && apiKeyValue != "" {
		s.apiKeyService.invalidateAPIKeyCaches(ctx, apiKeyValue)
	}
	return s.GetKey(ctx, id)
}

func (s *RelayAdminService) UpdateKeyLimits(ctx context.Context, id int, input RelayAdminKeyLimitsInput) (*RelayAdminKey, error) {
	limits := input.normalizedLimits()
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE relay_keys SET daily_request_limit = %s, daily_token_limit = %s, monthly_cost_limit = %s, concurrency_limit = %s, updated_at = CURRENT_TIMESTAMP WHERE id = %s AND deleted_at = 0`,
		relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2), relayPlaceholder(dialectName, 3), relayPlaceholder(dialectName, 4), relayPlaceholder(dialectName, 5))
	result, err := db.ExecContext(ctx, query, nullablePositiveInt64(limits.DailyRequestLimit), nullablePositiveInt64(limits.DailyTokenLimit), nullablePositiveDecimal(limits.MonthlyCostLimit), nullablePositiveInt64(limits.ConcurrencyLimit), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update relay key limits: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, fmt.Errorf("relay key %d not found", id)
	}
	return s.GetKey(ctx, id)
}

func (s *RelayAdminService) GetWallet(ctx context.Context, relayKeyID int) (*RelayAdminWallet, error) {
	wallets, err := s.listWallets(ctx, &relayKeyID, nil)
	if err != nil {
		return nil, err
	}
	if len(wallets) == 0 {
		return nil, fmt.Errorf("relay wallet for key %d not found", relayKeyID)
	}
	return &wallets[0], nil
}

func (s *RelayAdminService) ListLedgerEntries(ctx context.Context, relayKeyID int) ([]RelayAdminWalletLedgerEntry, error) {
	return s.listLedgerEntries(ctx, &relayKeyID, nil, nil)
}

func (s *RelayAdminService) RechargeWallet(ctx context.Context, input RelayAdminRechargeInput) (*RelayAdminWalletLedgerEntry, error) {
	relayKeyID, err := parseRelayAdminID(input.RelayKeyID, "relayKeyId")
	if err != nil {
		return nil, err
	}
	amount := decimal.NewFromFloat(input.Amount)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("recharge amount must be greater than 0")
	}
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin relay wallet recharge transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	selectQuery := fmt.Sprintf(`SELECT rk.project_id, COALESCE(rw.currency, 'USD'), COALESCE(rw.available_amount, '0')
FROM relay_keys rk
LEFT JOIN relay_wallets rw ON rw.relay_key_id = rk.id
WHERE rk.id = %s AND rk.deleted_at = 0`, relayPlaceholder(dialectName, 1))
	var (
		projectID    int
		currency     string
		availableRaw sql.NullString
	)
	if err := tx.QueryRowContext(ctx, selectQuery, relayKeyID).Scan(&projectID, &currency, &availableRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("relay key %d not found", relayKeyID)
		}
		return nil, fmt.Errorf("failed to load relay wallet: %w", err)
	}
	available, err := parseRelayDecimal(availableRaw, "wallet available amount")
	if err != nil {
		return nil, err
	}
	balanceAfter := available.Add(amount)
	// Use atomic increment to avoid lost-update under concurrency.
	updateQuery := fmt.Sprintf("UPDATE relay_wallets SET available_amount = available_amount + %s, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE relay_key_id = %s", relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2))
	result, err := tx.ExecContext(ctx, updateQuery, amount.String(), relayKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to update relay wallet: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		_, err = execRelayInsertTx(ctx, tx, dialectName,
			fmt.Sprintf("INSERT INTO relay_wallets (relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version) VALUES (%s)", strings.Join(relayPlaceholders(dialectName, 7, 1), ",")),
			relayKeyID, projectID, currency, amount.String(), "0", "0", 1,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create relay wallet: %w", err)
		}
	}
	operatorID := 0
	if user, ok := contexts.GetUser(ctx); ok && user != nil {
		operatorID = user.ID
	}
	ledgerID, err := insertRelayAdminLedgerTx(ctx, tx, dialectName, relayAdminLedgerInsert{
		RelayKeyID:     relayKeyID,
		ProjectID:      projectID,
		Direction:      "credit",
		Scene:          "recharge",
		Amount:         amount,
		BalanceBefore:  available,
		BalanceAfter:   balanceAfter,
		IdempotencyKey: fmt.Sprintf("manual_recharge:%d:%d", relayKeyID, time.Now().UnixNano()),
		OperatorUserID: operatorID,
		Remark:         strings.TrimSpace(input.Note),
	})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit relay wallet recharge: %w", err)
	}
	committed = true
	entries, err := s.listLedgerEntries(ctx, nil, &ledgerID, nil)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("relay wallet ledger entry %d not found", ledgerID)
	}
	return &entries[0], nil
}

func (s *RelayAdminService) ListRequestTraces(ctx context.Context, projectID *int) ([]RelayRequestTraceView, error) {
	return s.listRequestTraces(ctx, projectID, 100)
}

func (s *RelayAdminService) ListChannelPoolHealth(ctx context.Context) ([]RelayChannelPoolHealthView, error) {
	products, err := s.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RelayChannelPoolHealthView, 0, len(products))
	for _, product := range products {
		health := RelayChannelPoolHealthView{
			ProductID:   product.ID,
			ProductName: product.Name,
			Status:      product.PoolHealth,
			Channels:    product.ChannelPool,
		}
		for _, channel := range product.ChannelPool {
			switch channel.Health {
			case RelayHealthStatusHealthy:
				health.HealthyChannels++
			case RelayHealthStatusDegraded:
				health.DegradedChannels++
			default:
				health.UnavailableChannels++
				if health.RiskReason == "" {
					health.RiskReason = channel.UnavailableReason
				}
			}
			if health.RiskReason == "" && channel.UnavailableReason != "" {
				health.RiskReason = channel.UnavailableReason
			}
		}
		out = append(out, health)
	}
	return out, nil
}

func (s *RelayAdminService) GetOverview(ctx context.Context, projectID *int) (*ProjectRelayOverviewView, error) {
	products, err := s.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := s.ListKeys(ctx, projectID)
	if err != nil {
		return nil, err
	}
	wallets, err := s.listWallets(ctx, nil, projectID)
	if err != nil {
		return nil, err
	}
	requests, err := s.listRequestTraces(ctx, projectID, 5)
	if err != nil {
		return nil, err
	}
	usage, err := s.listUsageSummaries(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &ProjectRelayOverviewView{Products: products, Keys: keys, Wallets: wallets, RecentRequests: requests, Usage: usage}, nil
}

func (s *RelayAdminService) GetUsage(ctx context.Context, projectID *int) (*ProjectRelayUsageView, error) {
	wallets, err := s.listWallets(ctx, nil, projectID)
	if err != nil {
		return nil, err
	}
	ledger, err := s.listLedgerEntries(ctx, nil, nil, projectID)
	if err != nil {
		return nil, err
	}
	usage, err := s.listUsageSummaries(ctx, projectID)
	if err != nil {
		return nil, err
	}
	requests, err := s.listRequestTraces(ctx, projectID, 50)
	if err != nil {
		return nil, err
	}
	return &ProjectRelayUsageView{Wallets: wallets, LedgerEntries: ledger, Usage: usage, RecentRequests: requests}, nil
}

func (s *RelayAdminService) listProductRows(ctx context.Context) ([]RelayAdminProduct, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	query := `SELECT id, code, name, provider_type, access_mode, billing_mode, status, currency, allowed_models, request_timeout_seconds, list_price_config, created_at, updated_at
FROM relay_products
WHERE deleted_at = 0
ORDER BY id ASC`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminProduct{}, nil
		}
		return nil, fmt.Errorf("failed to list relay products: %w", err)
	}
	defer rows.Close()
	out := []RelayAdminProduct{}
	for rows.Next() {
		product, err := scanRelayAdminProduct(rows)
		if err != nil {
			return nil, err
		}
		if err := s.enrichProduct(ctx, db, dialectName, &product); err != nil {
			return nil, err
		}
		out = append(out, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) loadProductRow(ctx context.Context, id int) (*RelayAdminProduct, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT id, code, name, provider_type, access_mode, billing_mode, status, currency, allowed_models, request_timeout_seconds, list_price_config, created_at, updated_at
FROM relay_products
WHERE id = %s AND deleted_at = 0
LIMIT 1`, relayPlaceholder(dialectName, 1))
	product, err := scanRelayAdminProduct(db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("relay product %d not found", id)
		}
		return nil, err
	}
	if err := s.enrichProduct(ctx, db, dialectName, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (s *RelayAdminService) enrichProduct(ctx context.Context, db *sql.DB, dialectName string, product *RelayAdminProduct) error {
	id, err := strconv.Atoi(product.ID)
	if err != nil {
		return err
	}
	channels, err := s.listProductChannels(ctx, db, dialectName, id, product.ProviderType)
	if err != nil {
		return err
	}
	product.ChannelPool = channels
	product.PoolHealth = summarizeRelayPoolHealth(channels)
	stats, err := s.productStats(ctx, db, dialectName, id)
	if err != nil {
		return err
	}
	product.KeyCount = stats.keyCount
	product.ActiveKeyCount = stats.activeKeyCount
	product.MonthlyRequestCount = stats.monthlyRequestCount
	product.MonthlyTokenCount = stats.monthlyTokenCount
	product.MonthlyCost = stats.monthlyCost
	return nil
}

type relayProductStats struct {
	keyCount            int
	activeKeyCount      int
	monthlyRequestCount int64
	monthlyTokenCount   int64
	monthlyCost         float64
}

func (s *RelayAdminService) productStats(ctx context.Context, db *sql.DB, dialectName string, productID int) (relayProductStats, error) {
	var stats relayProductStats
	keyQuery := fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END), 0)
FROM relay_keys
WHERE product_id = %s AND deleted_at = 0`, relayPlaceholder(dialectName, 1))
	if err := db.QueryRowContext(ctx, keyQuery, productID).Scan(&stats.keyCount, &stats.activeKeyCount); err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product key stats: %w", err)
	}
	monthStart := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	chargeExpr := relayAdminDecimalCast(dialectName, "rdus.total_charge")
	usageQuery := fmt.Sprintf(`SELECT COALESCE(SUM(rdus.request_count), 0), COALESCE(SUM(rdus.total_tokens), 0), COALESCE(SUM(%s), 0)
FROM relay_daily_usage_summaries rdus
JOIN relay_keys rk ON rk.id = rdus.relay_key_id
WHERE rk.product_id = %s AND rdus.stat_date >= %s`, chargeExpr, relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2))
	if err := db.QueryRowContext(ctx, usageQuery, productID, monthStart).Scan(&stats.monthlyRequestCount, &stats.monthlyTokenCount, &stats.monthlyCost); err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product usage stats: %w", err)
	}
	return stats, nil
}

func (s *RelayAdminService) listProductChannels(ctx context.Context, db *sql.DB, dialectName string, productID int, provider RelayProductProviderType) ([]RelayAdminProductChannel, error) {
	query := fmt.Sprintf(`SELECT rpc.id, rpc.channel_id, COALESCE(c.name, ''), rpc.priority, rpc.weight, rpc.status, rpc.allow_fallback, rpc.model_filter, COALESCE(c.status, ''), COALESCE(c.error_message, ''), pqs.status, pqs.ready, pqs.next_check_at, rpc.updated_at
FROM relay_product_channels rpc
JOIN channels c ON c.id = rpc.channel_id AND c.deleted_at = 0
LEFT JOIN provider_quota_status pqs ON pqs.channel_id = c.id
WHERE rpc.product_id = %s
ORDER BY rpc.priority ASC, rpc.weight DESC, rpc.channel_id ASC`, relayPlaceholder(dialectName, 1))
	rows, err := db.QueryContext(ctx, query, productID)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminProductChannel{}, nil
		}
		return nil, fmt.Errorf("failed to list relay product channels: %w", err)
	}
	defer rows.Close()
	out := []RelayAdminProductChannel{}
	for rows.Next() {
		var (
			ch             RelayAdminProductChannel
			bindingID      int
			channelID      int
			bindingStatus  string
			modelFilterRaw sql.NullString
			channelStatus  string
			channelError   string
			quotaStatus    sql.NullString
			quotaReady     sql.NullBool
			nextCheckAt    relayNullTime
			bindingUpdated relayNullTime
		)
		if err := rows.Scan(&bindingID, &channelID, &ch.ChannelName, &ch.Priority, &ch.Weight, &bindingStatus, &ch.AllowFallback, &modelFilterRaw, &channelStatus, &channelError, &quotaStatus, &quotaReady, &nextCheckAt, &bindingUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan relay product channel: %w", err)
		}
		ch.ID = relayStringID(bindingID)
		ch.ChannelID = relayStringID(channelID)
		ch.Provider = provider
		ch.Status = RelayProductChannelStatus(bindingStatus)
		ch.ModelFilter = parseRelayModelFilter(modelFilterRaw)
		if ch.ModelFilter == nil {
			ch.ModelFilter = []string{}
		}
		ch.QuotaRemainingPercent = relayQuotaRemaining(quotaStatus, quotaReady)
		ch.Health, ch.UnavailableReason = relayChannelHealth(ch.Status, channelStatus, channelError, quotaStatus, quotaReady)
		ch.LastCheckedAt = nextCheckAt.String()
		if ch.LastCheckedAt == "" {
			ch.LastCheckedAt = bindingUpdated.String()
		}
		out = append(out, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) listKeys(ctx context.Context, projectID *int, relayKeyID int) ([]RelayAdminKey, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	clauses := []string{"rk.deleted_at = 0"}
	args := []any{}
	if projectID != nil {
		args = append(args, *projectID)
		clauses = append(clauses, fmt.Sprintf("rk.project_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	if relayKeyID > 0 {
		args = append(args, relayKeyID)
		clauses = append(clauses, fmt.Sprintf("rk.id = %s", relayPlaceholder(dialectName, len(args))))
	}
	today := relayUTCDate(time.Now())
	monthStart := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	args = append(args, today)
	todayPlaceholder := relayPlaceholder(dialectName, len(args))
	args = append(args, monthStart)
	monthPlaceholder := relayPlaceholder(dialectName, len(args))
	monthlyCostExpr := relayAdminDecimalCast(dialectName, "total_charge")
	query := fmt.Sprintf(`SELECT rk.id, rk.api_key_id, rk.project_id, COALESCE(p.name, ''), rk.product_id, rp.name, rk.display_name, ak.key, rk.status, rk.balance_mode, rk.expires_at, rk.created_at, rk.last_used_at, rk.daily_request_limit, rk.daily_token_limit, rk.monthly_cost_limit, rk.concurrency_limit, COALESCE(rdus.request_count, 0), COALESCE(rdus.total_tokens, 0), COALESCE(month_usage.monthly_cost, '0'), COALESCE(rw.available_amount, '0'), rp.provider_type
FROM relay_keys rk
JOIN api_keys ak ON ak.id = rk.api_key_id AND ak.deleted_at = 0
JOIN relay_products rp ON rp.id = rk.product_id AND rp.deleted_at = 0
LEFT JOIN projects p ON p.id = rk.project_id AND p.deleted_at = 0
LEFT JOIN relay_wallets rw ON rw.relay_key_id = rk.id
LEFT JOIN relay_daily_usage_summaries rdus ON rdus.relay_key_id = rk.id AND rdus.stat_date = %s
LEFT JOIN (
	SELECT relay_key_id, SUM(%s) AS monthly_cost
	FROM relay_daily_usage_summaries
	WHERE stat_date >= %s
	GROUP BY relay_key_id
) month_usage ON month_usage.relay_key_id = rk.id
WHERE %s
ORDER BY rk.id ASC`, todayPlaceholder, monthlyCostExpr, monthPlaceholder, strings.Join(clauses, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminKey{}, nil
		}
		return nil, fmt.Errorf("failed to list relay keys: %w", err)
	}
	defer rows.Close()
	out := []RelayAdminKey{}
	for rows.Next() {
		key, productProvider, walletAmount, err := scanRelayAdminKey(rows)
		if err != nil {
			return nil, err
		}
		key.BaseURL = relayBaseURL(productProvider)
		key.DerivedStates = s.deriveKeyStates(ctx, &key, walletAmount)
		out = append(out, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) deriveKeyStates(ctx context.Context, key *RelayAdminKey, walletAmount decimal.Decimal) []RelayDerivedState {
	states := []RelayDerivedState{}
	if key.ExpiresAt != "" {
		if expiresAt, err := time.Parse(time.RFC3339, key.ExpiresAt); err == nil && !expiresAt.After(time.Now()) {
			states = append(states, RelayDerivedStateExpired)
		}
	}
	if key.BalanceMode == RelayKeyBalanceModePrepaid && walletAmount.LessThanOrEqual(decimal.NewFromInt(10)) {
		states = append(states, RelayDerivedStateLowBalance)
	}
	if (key.Limits.DailyRequestLimit > 0 && key.Usage.TodayRequests >= key.Limits.DailyRequestLimit) || (key.Limits.DailyTokenLimit > 0 && key.Usage.TodayTokens >= key.Limits.DailyTokenLimit) || (key.Limits.MonthlyCostLimit > 0 && key.Usage.MonthlyCost >= key.Limits.MonthlyCostLimit) {
		states = append(states, RelayDerivedStateQuotaReached)
	}
	productID, err := strconv.Atoi(key.ProductID)
	if err != nil {
		log.Error(ctx, "failed to parse relay key product id", log.String("product_id", key.ProductID), log.Cause(err))
	} else {
		product, err := s.GetProduct(ctx, productID)
		if err != nil {
			log.Error(ctx, "failed to get relay product for key state derivation", log.Int("product_id", productID), log.Cause(err))
		} else if product.PoolHealth != RelayHealthStatusHealthy {
			states = append(states, RelayDerivedStateUpstreamPoolDegraded)
		}
	}
	return states
}

func (s *RelayAdminService) listWallets(ctx context.Context, relayKeyID *int, projectID *int) ([]RelayAdminWallet, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	clauses := []string{"1 = 1"}
	args := []any{}
	if relayKeyID != nil {
		args = append(args, *relayKeyID)
		clauses = append(clauses, fmt.Sprintf("rw.relay_key_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	if projectID != nil {
		args = append(args, *projectID)
		clauses = append(clauses, fmt.Sprintf("rw.project_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	ledgerAmountExpr := relayAdminDecimalCast(dialectName, "le.amount")
	query := fmt.Sprintf(`SELECT rw.id, rw.relay_key_id, rw.currency, rw.available_amount, rw.frozen_amount, rw.overdraft_limit, rw.updated_at,
COALESCE(SUM(CASE WHEN le.direction = 'credit' AND le.scene = 'recharge' THEN %s ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN le.direction = 'debit' THEN %s ELSE 0 END), 0)
FROM relay_wallets rw
LEFT JOIN relay_wallet_ledger_entries le ON le.relay_key_id = rw.relay_key_id
WHERE %s
GROUP BY rw.id, rw.relay_key_id, rw.currency, rw.available_amount, rw.frozen_amount, rw.overdraft_limit, rw.updated_at
ORDER BY rw.id ASC`, ledgerAmountExpr, ledgerAmountExpr, strings.Join(clauses, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminWallet{}, nil
		}
		return nil, fmt.Errorf("failed to list relay wallets: %w", err)
	}
	defer rows.Close()
	out := []RelayAdminWallet{}
	for rows.Next() {
		var (
			wallet         RelayAdminWallet
			availableRaw   sql.NullString
			frozenRaw      sql.NullString
			overdraftRaw   sql.NullString
			updated        relayNullTime
			totalRecharged sql.NullFloat64
			totalSpent     sql.NullFloat64
		)
		if err := rows.Scan(&wallet.ID, &wallet.RelayKeyID, &wallet.Currency, &availableRaw, &frozenRaw, &overdraftRaw, &updated, &totalRecharged, &totalSpent); err != nil {
			return nil, err
		}
		wallet.ID = relayNormalizeStringID(wallet.ID)
		wallet.RelayKeyID = relayNormalizeStringID(wallet.RelayKeyID)
		wallet.AvailableAmount = relayDecimalFloat(availableRaw)
		wallet.FrozenAmount = relayDecimalFloat(frozenRaw)
		wallet.CreditLimit = relayDecimalFloat(overdraftRaw)
		wallet.TotalRecharged = totalRecharged.Float64
		wallet.TotalSpent = totalSpent.Float64
		wallet.LowBalanceThreshold = 10
		wallet.UpdatedAt = updated.String()
		out = append(out, wallet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) listLedgerEntries(ctx context.Context, relayKeyID *int, ledgerID *int, projectID *int) ([]RelayAdminWalletLedgerEntry, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	clauses := []string{"1 = 1"}
	args := []any{}
	if relayKeyID != nil {
		args = append(args, *relayKeyID)
		clauses = append(clauses, fmt.Sprintf("le.relay_key_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	if ledgerID != nil {
		args = append(args, *ledgerID)
		clauses = append(clauses, fmt.Sprintf("le.id = %s", relayPlaceholder(dialectName, len(args))))
	}
	if projectID != nil {
		args = append(args, *projectID)
		clauses = append(clauses, fmt.Sprintf("rw.project_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	query := fmt.Sprintf(`SELECT le.id, le.relay_key_id, le.direction, le.scene, le.amount, COALESCE(rw.currency, 'USD'), le.balance_after, le.request_id, le.usage_log_id, le.operator_user_id, le.remark, le.created_at
FROM relay_wallet_ledger_entries le
LEFT JOIN relay_wallets rw ON rw.relay_key_id = le.relay_key_id
WHERE %s
ORDER BY le.created_at DESC, le.id DESC
LIMIT 200`, strings.Join(clauses, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminWalletLedgerEntry{}, nil
		}
		return nil, fmt.Errorf("failed to list relay wallet ledger entries: %w", err)
	}
	defer rows.Close()
	out := []RelayAdminWalletLedgerEntry{}
	for rows.Next() {
		entry, err := scanRelayAdminLedgerEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) listRequestTraces(ctx context.Context, projectID *int, limit int) ([]RelayRequestTraceView, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	clauses := []string{"rk.deleted_at = 0"}
	args := []any{}
	if projectID != nil {
		args = append(args, *projectID)
		clauses = append(clauses, fmt.Sprintf("rk.project_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	args = append(args, limit)
	limitPlaceholder := relayPlaceholder(dialectName, len(args))
	query := fmt.Sprintf(`SELECT r.id, COALESCE(r.external_id, ''), r.created_at, COALESCE(p.name, ''), rk.display_name, rp.name, r.model_id, COALESCE(c.name, ''), rp.provider_type, r.status, r.metrics_latency_ms, re.response_status_code, ul.id, ul.prompt_tokens, ul.completion_tokens, ul.total_cost, le.id, le.amount, re.error_message
FROM requests r
JOIN relay_keys rk ON rk.api_key_id = r.api_key_id
JOIN relay_products rp ON rp.id = rk.product_id
LEFT JOIN projects p ON p.id = rk.project_id AND p.deleted_at = 0
LEFT JOIN channels c ON c.id = r.channel_id AND c.deleted_at = 0
LEFT JOIN usage_logs ul ON ul.request_id = r.id
LEFT JOIN relay_wallet_ledger_entries le ON le.usage_log_id = ul.id
LEFT JOIN request_executions re ON re.request_id = r.id
WHERE %s
ORDER BY r.created_at DESC, r.id DESC
LIMIT %s`, strings.Join(clauses, " AND "), limitPlaceholder)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayRequestTraceView{}, nil
		}
		return nil, fmt.Errorf("failed to list relay request traces: %w", err)
	}
	defer rows.Close()
	out := []RelayRequestTraceView{}
	seen := map[string]struct{}{}
	for rows.Next() {
		trace, err := scanRelayRequestTrace(rows)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[trace.ID]; ok {
			continue
		}
		seen[trace.ID] = struct{}{}
		out = append(out, trace)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RelayAdminService) listUsageSummaries(ctx context.Context, projectID *int) ([]RelayDailyUsageSummaryView, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	clauses := []string{"1 = 1"}
	args := []any{}
	if projectID != nil {
		args = append(args, *projectID)
		clauses = append(clauses, fmt.Sprintf("rdus.project_id = %s", relayPlaceholder(dialectName, len(args))))
	}
	query := fmt.Sprintf(`SELECT rdus.relay_key_id, rdus.stat_date, rdus.request_count, rdus.total_tokens, rdus.total_charge
FROM relay_daily_usage_summaries rdus
WHERE %s
ORDER BY rdus.stat_date DESC, rdus.relay_key_id ASC
LIMIT 120`, strings.Join(clauses, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayDailyUsageSummaryView{}, nil
		}
		return nil, fmt.Errorf("failed to list relay usage summaries: %w", err)
	}
	defer rows.Close()
	out := []RelayDailyUsageSummaryView{}
	for rows.Next() {
		var (
			view        RelayDailyUsageSummaryView
			statDate    relayNullTime
			totalTokens sql.NullInt64
			totalCharge sql.NullString
		)
		if err := rows.Scan(&view.RelayKeyID, &statDate, &view.Requests, &totalTokens, &totalCharge); err != nil {
			return nil, err
		}
		view.RelayKeyID = relayNormalizeStringID(view.RelayKeyID)
		view.StatDate = statDate.DateString()
		if totalTokens.Valid {
			view.CompletionTokens = totalTokens.Int64
		}
		view.TotalCost = relayDecimalFloat(totalCharge)
		out = append(out, view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanRelayAdminProduct(row relayScanner) (RelayAdminProduct, error) {
	var (
		product          RelayAdminProduct
		productID        int
		providerType     string
		accessMode       string
		billingMode      string
		status           string
		allowedModelsRaw sql.NullString
		priceConfigRaw   sql.NullString
		createdAt        relayNullTime
		updatedAt        relayNullTime
	)
	if err := row.Scan(&productID, &product.Code, &product.Name, &providerType, &accessMode, &billingMode, &status, &product.Currency, &allowedModelsRaw, &product.RequestTimeoutSeconds, &priceConfigRaw, &createdAt, &updatedAt); err != nil {
		return product, err
	}
	models, err := parseRelayStringList(allowedModelsRaw)
	if err != nil {
		return product, err
	}
	priceConfig, err := parseRelayJSONObject(priceConfigRaw)
	if err != nil {
		return product, err
	}
	product.ID = relayStringID(productID)
	product.ProviderType = RelayProductProviderType(providerType)
	product.AccessMode = RelayProductAccessMode(accessMode)
	product.BillingMode = RelayProductBillingMode(billingMode)
	product.Status = RelayProductStatus(status)
	product.AllowedModels = models
	if product.AllowedModels == nil {
		product.AllowedModels = []string{}
	}
	product.ListPriceConfig = priceConfig
	if description, ok := priceConfig["description"].(string); ok {
		product.Description = description
	}
	product.DefaultTimeoutMs = product.RequestTimeoutSeconds * 1000
	product.CreatedAt = createdAt.String()
	product.UpdatedAt = updatedAt.String()
	return product, nil
}

func scanRelayAdminKey(row relayScanner) (RelayAdminKey, RelayProductProviderType, decimal.Decimal, error) {
	var (
		key               RelayAdminKey
		keyID             int
		apiKeyID          int
		projectID         int
		productID         int
		status            string
		balanceMode       string
		expiresAt         relayNullTime
		createdAt         relayNullTime
		lastUsedAt        relayNullTime
		dailyRequestLimit sql.NullInt64
		dailyTokenLimit   sql.NullInt64
		monthlyCostLimit  sql.NullString
		concurrencyLimit  sql.NullInt64
		monthlyCost       sql.NullString
		walletAmountRaw   sql.NullString
		apiKey            string
		providerType      string
	)
	if err := row.Scan(&keyID, &apiKeyID, &projectID, &key.ProjectName, &productID, &key.ProductName, &key.Name, &apiKey, &status, &balanceMode, &expiresAt, &createdAt, &lastUsedAt, &dailyRequestLimit, &dailyTokenLimit, &monthlyCostLimit, &concurrencyLimit, &key.Usage.TodayRequests, &key.Usage.TodayTokens, &monthlyCost, &walletAmountRaw, &providerType); err != nil {
		return key, "", decimal.Zero, err
	}
	key.ID = relayStringID(keyID)
	key.APIKeyID = relayStringID(apiKeyID)
	key.ProjectID = relayStringID(projectID)
	key.ProductID = relayStringID(productID)
	key.MaskedKey = maskRelayAPIKey(apiKey)
	key.Status = RelayKeyStatus(status)
	key.BalanceMode = RelayKeyBalanceMode(balanceMode)
	key.ExpiresAt = expiresAt.String()
	key.CreatedAt = createdAt.String()
	key.LastUsedAt = lastUsedAt.String()
	if dailyRequestLimit.Valid {
		key.Limits.DailyRequestLimit = dailyRequestLimit.Int64
	}
	if dailyTokenLimit.Valid {
		key.Limits.DailyTokenLimit = dailyTokenLimit.Int64
	}
	if concurrencyLimit.Valid {
		key.Limits.ConcurrencyLimit = concurrencyLimit.Int64
	}
	key.Limits.MonthlyCostLimit = relayDecimalFloat(monthlyCostLimit)
	key.Usage.MonthlyCost = relayDecimalFloat(monthlyCost)
	walletAmount, err := parseRelayDecimal(walletAmountRaw, "wallet available amount")
	if err != nil {
		return key, "", decimal.Zero, err
	}
	return key, RelayProductProviderType(providerType), walletAmount, nil
}

func scanRelayAdminLedgerEntry(row relayScanner) (RelayAdminWalletLedgerEntry, error) {
	var (
		entry           RelayAdminWalletLedgerEntry
		entryID         int
		relayKeyID      int
		direction       string
		scene           string
		amountRaw       sql.NullString
		balanceAfterRaw sql.NullString
		requestID       sql.NullInt64
		usageLogID      sql.NullInt64
		operatorUserID  sql.NullInt64
		note            sql.NullString
		createdAt       relayNullTime
	)
	if err := row.Scan(&entryID, &relayKeyID, &direction, &scene, &amountRaw, &entry.Currency, &balanceAfterRaw, &requestID, &usageLogID, &operatorUserID, &note, &createdAt); err != nil {
		return entry, err
	}
	entry.ID = relayStringID(entryID)
	entry.RelayKeyID = relayStringID(relayKeyID)
	entry.Type = relayLedgerType(scene)
	entry.Amount = relayLedgerSignedAmount(direction, relayDecimalFloat(amountRaw))
	entry.BalanceAfter = relayDecimalFloat(balanceAfterRaw)
	if requestID.Valid {
		entry.ReferenceID = relayStringID(int(requestID.Int64))
	} else if usageLogID.Valid {
		entry.ReferenceID = relayStringID(int(usageLogID.Int64))
	}
	entry.Operator = "system"
	if operatorUserID.Valid {
		entry.Operator = fmt.Sprintf("user:%d", operatorUserID.Int64)
	}
	entry.Note = note.String
	entry.CreatedAt = createdAt.String()
	return entry, nil
}

func scanRelayRequestTrace(row relayScanner) (RelayRequestTraceView, error) {
	var (
		trace             RelayRequestTraceView
		requestID         int
		requestExternalID string
		createdAt         relayNullTime
		providerType      string
		requestStatus     string
		latency           sql.NullInt64
		responseStatus    sql.NullInt64
		usageLogID        sql.NullInt64
		ledgerEntryID     sql.NullInt64
		totalCost         sql.NullFloat64
		ledgerAmount      sql.NullString
		errorMessage      sql.NullString
	)
	if err := row.Scan(&requestID, &requestExternalID, &createdAt, &trace.ProjectName, &trace.KeyName, &trace.ProductName, &trace.ModelID, &trace.ChannelName, &providerType, &requestStatus, &latency, &responseStatus, &usageLogID, &trace.PromptTokens, &trace.CompletionTokens, &totalCost, &ledgerEntryID, &ledgerAmount, &errorMessage); err != nil {
		return trace, err
	}
	trace.ID = relayStringID(requestID)
	trace.RequestID = requestExternalID
	if trace.RequestID == "" {
		trace.RequestID = trace.ID
	}
	trace.CreatedAt = createdAt.String()
	trace.Provider = RelayProductProviderType(providerType)
	trace.Status = relayRequestStatus(requestStatus)
	if latency.Valid {
		trace.LatencyMs = latency.Int64
	}
	if responseStatus.Valid {
		trace.ResponseStatusCode = int(responseStatus.Int64)
	}
	if usageLogID.Valid {
		trace.UsageLogID = relayStringID(int(usageLogID.Int64))
	}
	if ledgerEntryID.Valid {
		trace.LedgerEntryID = relayStringID(int(ledgerEntryID.Int64))
	}
	trace.Charged = trace.LedgerEntryID != ""
	trace.ChargeAmount = totalCost.Float64
	if trace.ChargeAmount == 0 && ledgerAmount.Valid {
		trace.ChargeAmount = math.Abs(relayDecimalFloat(ledgerAmount))
	}
	trace.SettlementStatus = relaySettlementStatus(trace.Charged, trace.ChargeAmount)
	trace.FailureStage = relayFailureStage(trace.Status, trace.ResponseStatusCode, trace.ChannelName, errorMessage.String)
	trace.ErrorMessage = errorMessage.String
	return trace, nil
}

type relayNullTime struct {
	Time  time.Time
	Valid bool
}

func (t *relayNullTime) Scan(value any) error {
	if value == nil {
		t.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		t.Valid = true
		return nil
	case string:
		return t.scanString(v)
	case []byte:
		return t.scanString(string(v))
	default:
		return fmt.Errorf("unsupported time value %T", value)
	}
}

func (t *relayNullTime) scanString(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		t.Valid = false
		return nil
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			t.Time = parsed
			t.Valid = true
			return nil
		}
	}
	return fmt.Errorf("invalid time value %q", value)
}

func (t relayNullTime) String() string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

func (t relayNullTime) DateString() string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format("2006-01-02")
}

type relayAdminLedgerInsert struct {
	RelayKeyID     int
	ProjectID      int
	Direction      string
	Scene          string
	Amount         decimal.Decimal
	BalanceBefore  decimal.Decimal
	BalanceAfter   decimal.Decimal
	IdempotencyKey string
	OperatorUserID int
	Remark         string
}

func insertRelayAdminLedgerTx(ctx context.Context, tx *sql.Tx, dialectName string, input relayAdminLedgerInsert) (int, error) {
	columns := []string{"relay_key_id", "project_id", "direction", "scene", "amount", "balance_before", "balance_after", "upstream_cost", "price_snapshot", "idempotency_key", "operator_user_id", "remark"}
	query := fmt.Sprintf("INSERT INTO relay_wallet_ledger_entries (%s) VALUES (%s)", strings.Join(columns, ","), strings.Join(relayPlaceholders(dialectName, len(columns), 1), ","))
	return execRelayInsertTx(ctx, tx, dialectName, query,
		input.RelayKeyID,
		input.ProjectID,
		input.Direction,
		input.Scene,
		input.Amount.String(),
		input.BalanceBefore.String(),
		input.BalanceAfter.String(),
		nil,
		[]byte("{}"),
		input.IdempotencyKey,
		nullableInt(input.OperatorUserID),
		input.Remark,
	)
}

func execRelayInsertTx(ctx context.Context, tx *sql.Tx, dialectName, query string, args ...any) (int, error) {
	if dialectName == dialect.Postgres {
		var id int
		if err := tx.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *RelayAdminService) relayKeyAPIKey(ctx context.Context, relayKeyID int) (int, string, error) {
	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return 0, "", err
	}
	query := fmt.Sprintf(`SELECT ak.id, ak.key
FROM relay_keys rk
JOIN api_keys ak ON ak.id = rk.api_key_id
WHERE rk.id = %s AND rk.deleted_at = 0
LIMIT 1`, relayPlaceholder(dialectName, 1))
	var apiKeyID int
	var apiKeyValue string
	if err := db.QueryRowContext(ctx, query, relayKeyID).Scan(&apiKeyID, &apiKeyValue); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", fmt.Errorf("relay key %d not found", relayKeyID)
		}
		return 0, "", err
	}
	return apiKeyID, apiKeyValue, nil
}

func (s *RelayAdminService) resolveProjectID(ctx context.Context, value string) (int, error) {
	if strings.TrimSpace(value) != "" {
		return parseRelayAdminID(value, "projectId")
	}
	if projectID, ok := contexts.GetProjectID(ctx); ok && projectID > 0 {
		return projectID, nil
	}
	return 0, fmt.Errorf("projectId is required")
}

func (input RelayAdminKeyLimitsInput) normalizedLimits() RelayKeyLimitSnapshot {
	if input.Limits != nil {
		return *input.Limits
	}
	limits := RelayKeyLimitSnapshot{}
	if input.DailyRequestLimit != nil {
		limits.DailyRequestLimit = *input.DailyRequestLimit
	}
	if input.DailyTokenLimit != nil {
		limits.DailyTokenLimit = *input.DailyTokenLimit
	}
	if input.MonthlyCostLimit != nil {
		limits.MonthlyCostLimit = *input.MonthlyCostLimit
	}
	if input.ConcurrencyLimit != nil {
		limits.ConcurrencyLimit = *input.ConcurrencyLimit
	}
	return limits
}

func parseRelayAdminID(value, field string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	if guid, err := objects.ParseGUID(value); err == nil {
		if guid.ID <= 0 {
			return 0, fmt.Errorf("%s must be greater than 0", field)
		}
		return guid.ID, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a numeric id or AxonHub GUID", field)
	}
	return id, nil
}

func relayStringID(id int) string {
	if id <= 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func relayNormalizeStringID(value string) string {
	return strings.TrimSpace(value)
}

func relayTimeoutSeconds(timeoutMs int) int {
	if timeoutMs <= 0 {
		return 0
	}
	return int(math.Ceil(float64(timeoutMs) / 1000.0))
}

func relayAdminDecimalCast(dialectName, expression string) string {
	if dialectName == dialect.MySQL {
		return fmt.Sprintf("CAST(%s AS DECIMAL(36,18))", expression)
	}
	return fmt.Sprintf("CAST(%s AS NUMERIC)", expression)
}

func nullablePositiveInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullablePositiveDecimal(value float64) any {
	if value <= 0 {
		return nil
	}
	return decimal.NewFromFloat(value).String()
}

func relayDecimalFloat(value sql.NullString) float64 {
	parsed, err := parseRelayDecimal(value, "decimal")
	if err != nil {
		return 0
	}
	out, _ := parsed.Float64()
	return out
}

func maskRelayAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 12 {
		return key[:relayMinInt(len(key), 4)] + "..."
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func relayBaseURL(provider RelayProductProviderType) string {
	switch provider {
	case RelayProductProviderTypeClaudeCode:
		return "/anthropic/v1"
	default:
		return "/v1"
	}
}

func summarizeRelayPoolHealth(channels []RelayAdminProductChannel) RelayHealthStatus {
	if len(channels) == 0 {
		return RelayHealthStatusUnavailable
	}
	healthy := 0
	degraded := 0
	for _, ch := range channels {
		switch ch.Health {
		case RelayHealthStatusHealthy:
			healthy++
		case RelayHealthStatusDegraded:
			degraded++
		}
	}
	if healthy == len(channels) {
		return RelayHealthStatusHealthy
	}
	if healthy > 0 || degraded > 0 {
		return RelayHealthStatusDegraded
	}
	return RelayHealthStatusUnavailable
}

func relayChannelHealth(bindingStatus RelayProductChannelStatus, channelStatus, channelError string, quotaStatus sql.NullString, quotaReady sql.NullBool) (RelayHealthStatus, string) {
	if bindingStatus != RelayProductChannelStatusActive {
		return RelayHealthStatusUnavailable, "binding is paused"
	}
	if channelStatus != "enabled" {
		if channelError != "" {
			return RelayHealthStatusUnavailable, channelError
		}
		return RelayHealthStatusUnavailable, "channel is not enabled"
	}
	if quotaStatus.Valid {
		switch quotaStatus.String {
		case "exhausted":
			return RelayHealthStatusUnavailable, "provider quota is exhausted"
		case "warning", "unknown":
			return RelayHealthStatusDegraded, "provider quota needs attention"
		}
	}
	if quotaReady.Valid && !quotaReady.Bool {
		return RelayHealthStatusUnavailable, "provider quota is not ready"
	}
	return RelayHealthStatusHealthy, ""
}

func relayQuotaRemaining(quotaStatus sql.NullString, quotaReady sql.NullBool) int {
	if quotaReady.Valid && !quotaReady.Bool {
		return 0
	}
	if !quotaStatus.Valid {
		return 100
	}
	switch quotaStatus.String {
	case "exhausted":
		return 0
	case "warning":
		return 25
	case "unknown":
		return 50
	default:
		return 100
	}
}

func relayLedgerType(scene string) RelayWalletLedgerType {
	switch scene {
	case "recharge":
		return RelayWalletLedgerTypeRecharge
	case "consume":
		return RelayWalletLedgerTypeCharge
	case "refund":
		return RelayWalletLedgerTypeRefund
	case "freeze":
		return RelayWalletLedgerTypeFreeze
	case "unfreeze":
		return RelayWalletLedgerTypeUnfreeze
	default:
		return RelayWalletLedgerTypeAdjustment
	}
}

func relayLedgerSignedAmount(direction string, amount float64) float64 {
	if direction == "debit" && amount > 0 {
		return -amount
	}
	return amount
}

func relayRequestStatus(status string) string {
	switch status {
	case "completed":
		return "completed"
	case "pending", "processing":
		return "processing"
	default:
		return "failed"
	}
}

func relayFailureStage(status string, responseStatus int, channelName string, message string) RelayFailureStage {
	if status == "completed" {
		return RelayFailureStageNone
	}
	if responseStatus == 401 || responseStatus == 403 {
		return RelayFailureStageAuth
	}
	if responseStatus == 402 || responseStatus == 429 {
		return RelayFailureStageKeyValidation
	}
	if channelName == "" {
		return RelayFailureStageRouting
	}
	if strings.Contains(strings.ToLower(message), "settlement") {
		return RelayFailureStageSettlement
	}
	return RelayFailureStageUpstream
}

func relaySettlementStatus(charged bool, chargeAmount float64) RelaySettlementStatus {
	if charged {
		return RelaySettlementStatusCharged
	}
	if chargeAmount > 0 {
		return RelaySettlementStatusDelayed
	}
	return RelaySettlementStatusSkipped
}

func relayMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
