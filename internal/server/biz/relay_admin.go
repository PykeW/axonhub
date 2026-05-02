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
	entsql "entgo.io/ent/dialect/sql"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/apikey"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/relaydailyusagesummary"
	"github.com/looplj/axonhub/internal/ent/relaykey"
	"github.com/looplj/axonhub/internal/ent/relayproduct"
	"github.com/looplj/axonhub/internal/ent/relayproductchannel"
	"github.com/looplj/axonhub/internal/ent/relaywallet"
	"github.com/looplj/axonhub/internal/ent/relaywalletledgerentry"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
)

type RelayAdminServiceParams struct {
	fx.In

	Ent            *ent.Client
	ProductService *RelayProductService
	APIKeyService  *APIKeyService
}

// Legacy Relay/Sub-Key admin surface for the older operator-managed product/key workflow.
// Keep maintenance-only changes here until Share/Use fully replaces relay products, relay keys, and relay wallets.
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
	ProductID             string                    `json:"productId,omitempty"`
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
	PlaintextKey  string                `json:"plaintextKey,omitempty"`
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

type RelayAdminChannelBindingUpdateInput struct {
	Priority      *int                       `json:"priority,omitempty"`
	Weight        *int                       `json:"weight,omitempty"`
	Status        *RelayProductChannelStatus `json:"status,omitempty"`
	ModelFilter   []string                   `json:"modelFilter,omitempty"`
	AllowFallback *bool                      `json:"allowFallback,omitempty"`
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
	binding, err := s.productService.CreateRelayProductChannelBinding(ctx, RelayProductChannelBindingInput{
		ProductID:     productID,
		ChannelID:     channelID,
		Priority:      input.Priority,
		Weight:        input.Weight,
		Status:        RelayProductChannelStatusActive,
		AllowFallback: &allowFallback,
		ModelFilter:   relayAdminModelFilterToMap(input.ModelFilter),
	})
	if err != nil {
		return nil, err
	}
	return s.getProductChannelBinding(ctx, binding.ProductID, binding.ID)
}

func (s *RelayAdminService) UpdateProductChannelBinding(ctx context.Context, id int, input RelayAdminChannelBindingUpdateInput) (*RelayAdminProductChannel, error) {
	if s.productService == nil {
		return nil, fmt.Errorf("relay product service is not configured")
	}
	update := RelayProductChannelBindingUpdateInput{
		Priority:      input.Priority,
		Weight:        input.Weight,
		Status:        input.Status,
		AllowFallback: input.AllowFallback,
	}
	if input.ModelFilter != nil {
		update.ModelFilter = relayAdminModelFilterToMap(input.ModelFilter)
	}
	binding, err := s.productService.UpdateRelayProductChannelBinding(ctx, id, update)
	if err != nil {
		return nil, err
	}
	return s.getProductChannelBinding(ctx, binding.ProductID, binding.ID)
}

func (s *RelayAdminService) DeleteProductChannelBinding(ctx context.Context, id int) error {
	if s.productService == nil {
		return fmt.Errorf("relay product service is not configured")
	}
	return s.productService.DeleteRelayProductChannelBinding(ctx, id)
}

func (s *RelayAdminService) getProductChannelBinding(ctx context.Context, productID int, bindingID int) (*RelayAdminProductChannel, error) {
	product, err := s.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	for _, ch := range product.ChannelPool {
		if ch.ID == relayStringID(bindingID) {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("relay product channel binding %d: %w", bindingID, ErrRelayBindingNotFound)
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
		return nil, fmt.Errorf("relay key %d: %w", id, ErrRelayKeyNotFound)
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
		return nil, fmt.Errorf("relay key name: %w", ErrRelayRequiredField)
	}
	balanceMode := input.BalanceMode
	if balanceMode == "" {
		balanceMode = RelayKeyBalanceModePrepaid
	}
	if balanceMode != RelayKeyBalanceModePrepaid && balanceMode != RelayKeyBalanceModeQuotaOnly {
		return nil, fmt.Errorf("unsupported relay key balance mode %q: %w", balanceMode, ErrRelayUnsupportedValue)
	}
	if input.InitialBalance < 0 {
		return nil, fmt.Errorf("initialBalance cannot be negative")
	}
	if s.productService == nil {
		return nil, fmt.Errorf("relay product service is not configured")
	}
	if _, err := s.productService.getRelayProduct(ctx, productID); err != nil {
		return nil, err
	}
	apiKeyValue, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	userID := 0
	if user, ok := contexts.GetUser(ctx); ok && user != nil && user.ID > 0 {
		userID = user.ID
	}

	tx, err := s.entFromContext(ctx).Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin relay key transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	apiKeyCreate := tx.APIKey.Create().
		SetKey(apiKeyValue).
		SetName(name).
		SetType(apikey.TypeUser).
		SetStatus(apikey.StatusEnabled).
		SetScopes([]string{"read_channels", "write_requests"}).
		SetProfiles(&objects.APIKeyProfiles{}).
		SetProjectID(projectID)
	if userID > 0 {
		apiKeyCreate = apiKeyCreate.SetUserID(userID)
	}
	apiKey, err := apiKeyCreate.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create backing api key: %w", err)
	}

	var expiresAtPtr *time.Time
	if strings.TrimSpace(input.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.ExpiresAt))
		if err != nil {
			return nil, fmt.Errorf("invalid expiresAt: %w", errors.Join(ErrRelayInvalidInput, err))
		}
		expiresAtPtr = &parsed
	}
	var dailyRequestLimit *int64
	if input.Limits.DailyRequestLimit > 0 {
		dailyRequestLimit = &input.Limits.DailyRequestLimit
	}
	var dailyTokenLimit *int64
	if input.Limits.DailyTokenLimit > 0 {
		dailyTokenLimit = &input.Limits.DailyTokenLimit
	}
	var monthlyCostLimit *string
	if input.Limits.MonthlyCostLimit > 0 {
		value := decimal.NewFromFloat(input.Limits.MonthlyCostLimit).String()
		monthlyCostLimit = &value
	}
	var concurrencyLimit *int64
	if input.Limits.ConcurrencyLimit > 0 {
		concurrencyLimit = &input.Limits.ConcurrencyLimit
	}

	relayKey, err := tx.RelayKey.Create().
		SetAPIKeyID(apiKey.ID).
		SetProjectID(projectID).
		SetProductID(productID).
		SetDisplayName(name).
		SetStatus(relaykey.StatusActive).
		SetBalanceMode(relaykey.BalanceMode(balanceMode)).
		SetNillableDailyRequestLimit(dailyRequestLimit).
		SetNillableDailyTokenLimit(dailyTokenLimit).
		SetNillableMonthlyCostLimit(monthlyCostLimit).
		SetNillableConcurrencyLimit(concurrencyLimit).
		SetNillableExpiresAt(expiresAtPtr).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay key: %w", err)
	}

	initialBalance := decimal.NewFromFloat(input.InitialBalance)
	_, err = tx.RelayWallet.Create().
		SetRelayKeyID(relayKey.ID).
		SetProjectID(projectID).
		SetCurrency(RelayProductDefaultCurrency).
		SetAvailableAmount(initialBalance.String()).
		SetFrozenAmount("0").
		SetOverdraftLimit("0").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay wallet: %w", err)
	}

	if initialBalance.GreaterThan(decimal.Zero) {
		operatorID := 0
		if user, ok := contexts.GetUser(ctx); ok && user != nil {
			operatorID = user.ID
		}
		ledgerCreate := tx.RelayWalletLedgerEntry.Create().
			SetRelayKeyID(relayKey.ID).
			SetProjectID(projectID).
			SetDirection(relaywalletledgerentry.DirectionCredit).
			SetScene(relaywalletledgerentry.SceneRecharge).
			SetAmount(initialBalance.String()).
			SetBalanceBefore(decimal.Zero.String()).
			SetBalanceAfter(initialBalance.String()).
			SetIdempotencyKey(fmt.Sprintf("initial_recharge:%d", relayKey.ID)).
			SetRemark("initial balance")
		if operatorID > 0 {
			ledgerCreate = ledgerCreate.SetOperatorUserID(operatorID)
		}
		if _, err = ledgerCreate.Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to create relay wallet ledger entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit relay key transaction: %w", err)
	}
	committed = true
	if s.apiKeyService != nil {
		s.apiKeyService.invalidateAPIKeyCaches(ctx, apiKeyValue)
	}
	key, err := s.GetKey(ctx, relayKey.ID)
	if err != nil {
		return nil, err
	}
	key.PlaintextKey = apiKeyValue
	return key, nil
}

func (s *RelayAdminService) UpdateKeyStatus(ctx context.Context, id int, input RelayAdminKeyStatusInput) (*RelayAdminKey, error) {
	if input.Status != RelayKeyStatusActive && input.Status != RelayKeyStatusSuspended && input.Status != RelayKeyStatusExhausted && input.Status != RelayKeyStatusArchived {
		return nil, fmt.Errorf("unsupported relay key status %q: %w", input.Status, ErrRelayUnsupportedValue)
	}
	apiKeyID, apiKeyValue, err := s.relayKeyAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	tx, err := s.entFromContext(ctx).Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin update key status transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.RelayKey.UpdateOneID(id).
		SetStatus(relaykey.Status(input.Status)).
		Where(relaykey.DeletedAt(0)).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay key %d: %w", id, ErrRelayKeyNotFound)
		}
		return nil, fmt.Errorf("failed to update relay key status: %w", err)
	}

	apiStatus := apikey.StatusDisabled
	if input.Status == RelayKeyStatusActive {
		apiStatus = apikey.StatusEnabled
	} else if input.Status == RelayKeyStatusArchived {
		apiStatus = apikey.StatusArchived
	}
	if _, err := tx.APIKey.UpdateOneID(apiKeyID).SetStatus(apiStatus).Save(ctx); err != nil {
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
	client := s.entFromContext(ctx)
	u := client.RelayKey.UpdateOneID(id).Where(relaykey.DeletedAt(0))
	if limits.DailyRequestLimit > 0 {
		u.SetDailyRequestLimit(limits.DailyRequestLimit)
	} else {
		u.ClearDailyRequestLimit()
	}
	if limits.DailyTokenLimit > 0 {
		u.SetDailyTokenLimit(limits.DailyTokenLimit)
	} else {
		u.ClearDailyTokenLimit()
	}
	if limits.MonthlyCostLimit > 0 {
		u.SetMonthlyCostLimit(decimal.NewFromFloat(limits.MonthlyCostLimit).String())
	} else {
		u.ClearMonthlyCostLimit()
	}
	if limits.ConcurrencyLimit > 0 {
		u.SetConcurrencyLimit(limits.ConcurrencyLimit)
	} else {
		u.ClearConcurrencyLimit()
	}
	if _, err := u.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay key %d: %w", id, ErrRelayKeyNotFound)
		}
		return nil, fmt.Errorf("failed to update relay key limits: %w", err)
	}
	return s.GetKey(ctx, id)
}

func (s *RelayAdminService) GetWallet(ctx context.Context, relayKeyID int) (*RelayAdminWallet, error) {
	wallets, err := s.listWallets(ctx, &relayKeyID, nil)
	if err != nil {
		return nil, err
	}
	if len(wallets) == 0 {
		return nil, fmt.Errorf("relay wallet for key %d: %w", relayKeyID, ErrRelayWalletNotFound)
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
	if math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		return nil, fmt.Errorf("recharge amount must be finite")
	}
	amount := decimal.NewFromFloat(input.Amount)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("recharge amount must be greater than 0")
	}

	tx, err := s.entFromContext(ctx).Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin relay wallet recharge transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	relayKey, err := tx.RelayKey.Query().Where(relaykey.ID(relayKeyID), relaykey.DeletedAt(0)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay key %d: %w", relayKeyID, ErrRelayKeyNotFound)
		}
		return nil, fmt.Errorf("failed to load relay wallet: %w", err)
	}
	projectID := relayKey.ProjectID

	available := decimal.Zero
	balanceAfter := amount
	wallet, err := tx.RelayWallet.Query().Where(relaywallet.RelayKeyID(relayKeyID)).First(ctx)
	if ent.IsNotFound(err) {
		_, err = tx.RelayWallet.Create().
			SetRelayKeyID(relayKeyID).
			SetProjectID(projectID).
			SetCurrency(RelayProductDefaultCurrency).
			SetAvailableAmount(amount.String()).
			SetFrozenAmount("0").
			SetOverdraftLimit("0").
			SetVersion(1).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create relay wallet: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to load relay wallet: %w", err)
	} else {
		available, err = decimal.NewFromString(wallet.AvailableAmount)
		if err != nil {
			return nil, fmt.Errorf("invalid wallet available amount: %w", err)
		}
		balanceAfter = available.Add(amount)
		_, err = tx.RelayWallet.UpdateOneID(wallet.ID).
			SetAvailableAmount(balanceAfter.String()).
			SetVersion(wallet.Version + 1).
			Where(relaywallet.VersionEQ(wallet.Version)).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update relay wallet: %w", err)
		}
	}

	operatorID := 0
	if user, ok := contexts.GetUser(ctx); ok && user != nil {
		operatorID = user.ID
	}
	ledgerCreate := tx.RelayWalletLedgerEntry.Create().
		SetRelayKeyID(relayKeyID).
		SetProjectID(projectID).
		SetDirection(relaywalletledgerentry.DirectionCredit).
		SetScene(relaywalletledgerentry.SceneRecharge).
		SetAmount(amount.String()).
		SetBalanceBefore(available.String()).
		SetBalanceAfter(balanceAfter.String()).
		SetIdempotencyKey(fmt.Sprintf("manual_recharge:%d:%d", relayKeyID, time.Now().UnixNano())).
		SetRemark(strings.TrimSpace(input.Note))
	if operatorID > 0 {
		ledgerCreate = ledgerCreate.SetOperatorUserID(operatorID)
	}
	ledgerEntry, err := ledgerCreate.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay wallet ledger entry: %w", err)
	}
	if relayKey.Status == relaykey.StatusExhausted && balanceAfter.GreaterThan(decimal.Zero) {
		if _, err := tx.RelayKey.UpdateOneID(relayKey.ID).
			Where(relaykey.DeletedAt(0)).
			SetStatus(relaykey.StatusActive).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to restore exhausted relay key: %w", err)
		}
		apiKey, err := tx.APIKey.Query().Where(apikey.ID(relayKey.APIKeyID)).First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load backing api key: %w", err)
		}
		if _, err := tx.APIKey.UpdateOneID(apiKey.ID).SetStatus(apikey.StatusEnabled).Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to restore backing api key status: %w", err)
		}
		if s.apiKeyService != nil && apiKey.Key != "" {
			defer s.apiKeyService.invalidateAPIKeyCaches(ctx, apiKey.Key)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit relay wallet recharge: %w", err)
	}
	committed = true
	entries, err := s.listLedgerEntries(ctx, nil, &ledgerEntry.ID, nil)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("relay wallet ledger entry %d: %w", ledgerEntry.ID, ErrRelayLedgerNotFound)
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
	keys, err := s.ListKeys(ctx, projectID)
	if err != nil {
		return nil, err
	}
	products, err := s.projectScopedProducts(ctx, projectID, keys)
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

func (s *RelayAdminService) projectScopedProducts(ctx context.Context, projectID *int, keys []RelayAdminKey) ([]RelayAdminProduct, error) {
	if projectID == nil {
		return s.ListProducts(ctx)
	}
	if len(keys) == 0 {
		return []RelayAdminProduct{}, nil
	}

	productIDs := make(map[string]struct{}, len(keys))
	out := make([]RelayAdminProduct, 0, len(keys))
	for _, key := range keys {
		if _, exists := productIDs[key.ProductID]; exists {
			continue
		}
		productIDs[key.ProductID] = struct{}{}
		id, err := strconv.Atoi(key.ProductID)
		if err != nil {
			return nil, fmt.Errorf("invalid relay product id %q for project overview: %w", key.ProductID, err)
		}
		product, err := s.entFromContext(ctx).RelayProduct.Query().
			Where(relayproduct.ID(id), relayproduct.DeletedAt(0)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
			}
			return nil, err
		}
		out = append(out, relayAdminProductFromEnt(product))
	}
	return out, nil
}

func (s *RelayAdminService) listProductRows(ctx context.Context) ([]RelayAdminProduct, error) {
	rows, err := s.entFromContext(ctx).RelayProduct.Query().
		Where(relayproduct.DeletedAt(0)).
		Order(relayproduct.ByID(entsql.OrderAsc())).
		All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminProduct{}, nil
		}
		return nil, fmt.Errorf("failed to list relay products: %w", err)
	}

	out := make([]RelayAdminProduct, 0, len(rows))
	for _, row := range rows {
		product := relayAdminProductFromEnt(row)
		if err := s.enrichProduct(ctx, &product); err != nil {
			return nil, err
		}
		out = append(out, product)
	}
	return out, nil
}

func (s *RelayAdminService) loadProductRow(ctx context.Context, id int) (*RelayAdminProduct, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	r, err := s.entFromContext(ctx).RelayProduct.Query().
		Where(relayproduct.ID(id), relayproduct.DeletedAt(0)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
		}
		return nil, err
	}
	product := relayAdminProductFromEnt(r)
	if err := s.enrichProduct(ctx, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (s *RelayAdminService) enrichProduct(ctx context.Context, product *RelayAdminProduct) error {
	id, err := strconv.Atoi(product.ID)
	if err != nil {
		return err
	}
	channels, err := s.listProductChannels(ctx, id, product.ProviderType)
	if err != nil {
		return err
	}
	product.ChannelPool = channels
	product.PoolHealth = summarizeRelayPoolHealth(channels)
	stats, err := s.productStats(ctx, id)
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

func (s *RelayAdminService) productStats(ctx context.Context, productID int) (relayProductStats, error) {
	var stats relayProductStats
	client := s.entFromContext(ctx)

	keyCount, err := client.RelayKey.Query().
		Where(relaykey.ProductID(productID), relaykey.DeletedAt(0)).
		Count(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product key stats: %w", err)
	}
	stats.keyCount = keyCount

	activeKeyCount, err := client.RelayKey.Query().
		Where(relaykey.ProductID(productID), relaykey.DeletedAt(0), relaykey.StatusEQ(relaykey.Status(RelayKeyStatusActive))).
		Count(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product key stats: %w", err)
	}
	stats.activeKeyCount = activeKeyCount

	monthStart := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	keys, err := client.RelayKey.Query().
		Where(relaykey.ProductID(productID), relaykey.DeletedAt(0)).
		Select(relaykey.FieldID).
		Ints(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product key stats: %w", err)
	}
	if len(keys) == 0 {
		return stats, nil
	}

	summaries, err := client.RelayDailyUsageSummary.Query().
		Where(relaydailyusagesummary.RelayKeyIDIn(keys...), relaydailyusagesummary.StatDateGTE(monthStart)).
		All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return stats, nil
		}
		return stats, fmt.Errorf("failed to load relay product usage stats: %w", err)
	}
	for _, sum := range summaries {
		stats.monthlyRequestCount += sum.RequestCount
		stats.monthlyTokenCount += sum.TotalTokens
		charge, _ := decimal.NewFromString(sum.TotalCharge)
		stats.monthlyCost += charge.InexactFloat64()
	}
	return stats, nil
}

func (s *RelayAdminService) listProductChannels(ctx context.Context, productID int, provider RelayProductProviderType) ([]RelayAdminProductChannel, error) {
	bindings, err := s.entFromContext(ctx).RelayProductChannel.Query().
		Where(relayproductchannel.ProductID(productID)).
		WithChannel(func(q *ent.ChannelQuery) {
			q.Where(channel.DeletedAt(0)).WithProviderQuotaStatus()
		}).
		Order(
			relayproductchannel.ByPriority(entsql.OrderAsc()),
			relayproductchannel.ByWeight(entsql.OrderDesc()),
			relayproductchannel.ByChannelID(entsql.OrderAsc()),
		).
		All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminProductChannel{}, nil
		}
		return nil, fmt.Errorf("failed to list relay product channels: %w", err)
	}

	out := make([]RelayAdminProductChannel, 0, len(bindings))
	for _, b := range bindings {
		ch := RelayAdminProductChannel{
			ID:            relayStringID(b.ID),
			ProductID:     relayStringID(b.ProductID),
			ChannelID:     relayStringID(b.ChannelID),
			Provider:      provider,
			Priority:      b.Priority,
			Weight:        b.Weight,
			Status:        RelayProductChannelStatus(b.Status),
			AllowFallback: b.AllowFallback,
			ModelFilter:   relayAdminModelFilterFromMap(b.ModelFilter),
		}
		channelStatus := ""
		channelError := ""
		quotaStatus := sql.NullString{}
		quotaReady := sql.NullBool{}

		if c, err := b.Edges.ChannelOrErr(); err == nil {
			ch.ChannelName = c.Name
			channelStatus = string(c.Status)
			if c.ErrorMessage != nil {
				channelError = *c.ErrorMessage
			}
			if pqs, err := c.Edges.ProviderQuotaStatusOrErr(); err == nil {
				quotaStatus = sql.NullString{String: string(pqs.Status), Valid: true}
				quotaReady = sql.NullBool{Bool: pqs.Ready, Valid: true}
				if !pqs.NextCheckAt.IsZero() {
					ch.LastCheckedAt = pqs.NextCheckAt.Format(time.RFC3339)
				}
			} else if !ent.IsNotFound(err) {
				var notLoaded *ent.NotLoadedError
				if !errors.As(err, &notLoaded) {
					return nil, err
				}
			}
		} else if !ent.IsNotFound(err) {
			var notLoaded *ent.NotLoadedError
			if !errors.As(err, &notLoaded) {
				return nil, err
			}
		}
		if ch.LastCheckedAt == "" {
			ch.LastCheckedAt = b.UpdatedAt.Format(time.RFC3339)
		}
		ch.QuotaRemainingPercent = relayQuotaRemaining(quotaStatus, quotaReady)
		ch.Health, ch.UnavailableReason = relayChannelHealth(ch.Status, channelStatus, channelError, quotaStatus, quotaReady)
		out = append(out, ch)
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
	client := s.entFromContext(ctx)
	q := client.RelayWalletLedgerEntry.Query()
	if relayKeyID != nil {
		q = q.Where(relaywalletledgerentry.RelayKeyID(*relayKeyID))
	}
	if ledgerID != nil {
		q = q.Where(relaywalletledgerentry.ID(*ledgerID))
	}
	if projectID != nil {
		q = q.Where(relaywalletledgerentry.ProjectID(*projectID))
	}
	entries, err := q.Order(
		relaywalletledgerentry.ByCreatedAt(entsql.OrderDesc()),
		relaywalletledgerentry.ByID(entsql.OrderDesc()),
	).Limit(200).All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminWalletLedgerEntry{}, nil
		}
		return nil, fmt.Errorf("failed to list relay wallet ledger entries: %w", err)
	}
	if len(entries) == 0 {
		return []RelayAdminWalletLedgerEntry{}, nil
	}

	relayKeyIDs := make([]int, 0, len(entries))
	seenRelayKeyIDs := make(map[int]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seenRelayKeyIDs[entry.RelayKeyID]; ok {
			continue
		}
		seenRelayKeyIDs[entry.RelayKeyID] = struct{}{}
		relayKeyIDs = append(relayKeyIDs, entry.RelayKeyID)
	}
	wallets, err := client.RelayWallet.Query().Where(relaywallet.RelayKeyIDIn(relayKeyIDs...)).All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayAdminWalletLedgerEntry{}, nil
		}
		return nil, fmt.Errorf("failed to list relay wallets for ledger entries: %w", err)
	}
	currenciesByRelayKeyID := make(map[int]string, len(wallets))
	for _, wallet := range wallets {
		currenciesByRelayKeyID[wallet.RelayKeyID] = wallet.Currency
	}

	out := make([]RelayAdminWalletLedgerEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, relayAdminLedgerEntryFromEnt(entry, currenciesByRelayKeyID[entry.RelayKeyID]))
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
	q := s.entFromContext(ctx).RelayDailyUsageSummary.Query()
	if projectID != nil {
		q = q.Where(relaydailyusagesummary.ProjectID(*projectID))
	}
	rows, err := q.Order(
		relaydailyusagesummary.ByStatDate(entsql.OrderDesc()),
		relaydailyusagesummary.ByRelayKeyID(entsql.OrderAsc()),
	).Limit(120).All(ctx)
	if err != nil {
		if isMissingRelayTableError(err) {
			return []RelayDailyUsageSummaryView{}, nil
		}
		return nil, fmt.Errorf("failed to list relay usage summaries: %w", err)
	}
	out := make([]RelayDailyUsageSummaryView, 0, len(rows))
	for _, row := range rows {
		view := RelayDailyUsageSummaryView{
			RelayKeyID:       relayStringID(row.RelayKeyID),
			StatDate:         row.StatDate.UTC().Format("2006-01-02"),
			Requests:         row.RequestCount,
			CompletionTokens: row.TotalTokens,
			TotalCost:        relayDecimalFloat(sql.NullString{String: row.TotalCharge, Valid: row.TotalCharge != ""}),
		}
		out = append(out, view)
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

func relayAdminProductFromEnt(e *ent.RelayProduct) RelayAdminProduct {
	models := e.AllowedModels
	if models == nil {
		models = []string{}
	}
	priceConfig := e.ListPriceConfig
	if priceConfig == nil {
		priceConfig = map[string]any{}
	}
	product := RelayAdminProduct{
		ID:                    relayStringID(e.ID),
		Code:                  e.Code,
		Name:                  e.Name,
		ProviderType:          RelayProductProviderType(e.ProviderType),
		AccessMode:            RelayProductAccessMode(e.AccessMode),
		BillingMode:           RelayProductBillingMode(e.BillingMode),
		Status:                RelayProductStatus(e.Status),
		Currency:              e.Currency,
		AllowedModels:         models,
		DefaultTimeoutMs:      e.RequestTimeoutSeconds * 1000,
		RequestTimeoutSeconds: e.RequestTimeoutSeconds,
		ListPriceConfig:       priceConfig,
		CreatedAt:             e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             e.UpdatedAt.Format(time.RFC3339),
	}
	if description, ok := priceConfig["description"].(string); ok {
		product.Description = description
	}
	return product
}

func relayAdminModelFilterToMap(models []string) map[string]any {
	modelFilter := map[string]any{}
	if models != nil {
		clean := make([]string, 0, len(models))
		for _, model := range models {
			if trimmed := strings.TrimSpace(model); trimmed != "" {
				clean = append(clean, trimmed)
			}
		}
		if len(clean) > 0 {
			modelFilter["models"] = clean
		}
	}
	return modelFilter
}

func relayAdminModelFilterFromMap(m any) []string {
	mf, ok := m.(map[string]any)
	if !ok || mf == nil || len(mf) == 0 {
		return []string{}
	}
	for _, key := range []string{"models", "allowedModels", "patterns", "include"} {
		if raw, ok := mf[key]; ok {
			switch v := raw.(type) {
			case []string:
				return v
			case []interface{}:
				out := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok {
						out = append(out, s)
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
	}
	return []string{}
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

func relayAdminLedgerEntryFromEnt(e *ent.RelayWalletLedgerEntry, currency string) RelayAdminWalletLedgerEntry {
	if currency == "" {
		currency = "USD"
	}
	entry := RelayAdminWalletLedgerEntry{
		ID:           relayStringID(e.ID),
		RelayKeyID:   relayStringID(e.RelayKeyID),
		Currency:     currency,
		Type:         relayLedgerType(e.Scene.String()),
		Amount:       relayLedgerSignedAmount(e.Direction.String(), relayDecimalFloat(sql.NullString{String: e.Amount, Valid: e.Amount != ""})),
		BalanceAfter: relayDecimalFloat(sql.NullString{String: e.BalanceAfter, Valid: e.BalanceAfter != ""}),
		Operator:     "system",
		CreatedAt:    e.CreatedAt.UTC().Format(time.RFC3339),
	}
	if e.RequestID != nil {
		entry.ReferenceID = relayStringID(*e.RequestID)
	} else if e.UsageLogID != nil {
		entry.ReferenceID = relayStringID(*e.UsageLogID)
	}
	if e.OperatorUserID != nil {
		entry.Operator = fmt.Sprintf("user:%d", *e.OperatorUserID)
	}
	if e.Remark != nil {
		entry.Note = *e.Remark
	}
	return entry
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
	relayKey, err := s.entFromContext(ctx).RelayKey.Query().Where(relaykey.ID(relayKeyID), relaykey.DeletedAt(0)).WithAPIKey().First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, "", fmt.Errorf("relay key %d: %w", relayKeyID, ErrRelayKeyNotFound)
		}
		return 0, "", err
	}
	apiKey, err := relayKey.Edges.APIKeyOrErr()
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, "", fmt.Errorf("relay key %d: %w", relayKeyID, ErrRelayKeyNotFound)
		}
		return 0, "", err
	}
	return apiKey.ID, apiKey.Key, nil
}

func (s *RelayAdminService) resolveProjectID(ctx context.Context, value string) (int, error) {
	if strings.TrimSpace(value) != "" {
		return parseRelayAdminID(value, "projectId")
	}
	if projectID, ok := contexts.GetProjectID(ctx); ok && projectID > 0 {
		return projectID, nil
	}
	return 0, fmt.Errorf("projectId: %w", ErrRelayRequiredField)
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
		return 0, fmt.Errorf("%s: %w", field, ErrRelayRequiredField)
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
