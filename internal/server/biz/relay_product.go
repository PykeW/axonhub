package biz

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
)

type RelayProductProviderType string

type RelayProductAccessMode string

type RelayProductBillingMode string

type RelayProductStatus string

type RelayProductChannelStatus string

const (
	RelayProductProviderTypeClaudeCode       RelayProductProviderType = "claudecode"
	RelayProductProviderTypeCodex            RelayProductProviderType = "codex"
	RelayProductProviderTypeOpenAICompatible RelayProductProviderType = "openai_compatible"
)

const (
	RelayProductAccessModeSharedCapacity RelayProductAccessMode = "shared_capacity"
)

const (
	RelayProductBillingModePrepaid   RelayProductBillingMode = "prepaid"
	RelayProductBillingModeQuotaOnly RelayProductBillingMode = "quota_only"
)

const (
	RelayProductStatusDraft    RelayProductStatus = "draft"
	RelayProductStatusActive   RelayProductStatus = "active"
	RelayProductStatusArchived RelayProductStatus = "archived"
)

const (
	RelayProductChannelStatusActive RelayProductChannelStatus = "active"
	RelayProductChannelStatusPaused RelayProductChannelStatus = "paused"
)

const (
	RelayProductDefaultCurrency              = "USD"
	RelayProductDefaultRequestTimeoutSeconds = 600
	RelayProductChannelDefaultWeight         = 100
)

type RelayProductServiceParams struct {
	fx.In

	Ent            *ent.Client
	ChannelService *ChannelService
}

type RelayProductService struct {
	*AbstractService

	channelService *ChannelService
}

type RelayProductContract struct {
	ProviderTypes []RelayProductProviderType   `json:"providerTypes"`
	AccessModes   []RelayProductAccessMode     `json:"accessModes"`
	BillingModes  []RelayProductBillingMode    `json:"billingModes"`
	Statuses      []RelayProductStatus         `json:"statuses"`
	BindingStatus []RelayProductChannelStatus  `json:"bindingStatus"`
	Defaults      RelayProductContractDefaults `json:"defaults"`
}

type RelayProductContractDefaults struct {
	Currency              string `json:"currency"`
	AccessMode            string `json:"accessMode"`
	BillingMode           string `json:"billingMode"`
	Status                string `json:"status"`
	RequestTimeoutSeconds int    `json:"requestTimeoutSeconds"`
	BindingWeight         int    `json:"bindingWeight"`
}

const (
	RelayProductTableName                 = "relay_products"
	RelayProductChannelTableName          = "relay_product_channels"
	RelayKeyTableName                     = "relay_keys"
	RelayWalletTableName                  = "relay_wallets"
	RelayWalletLedgerEntryTableName       = "relay_wallet_ledger_entries"
	RelayDailyUsageSummaryTableName       = "relay_daily_usage_summaries"
	RelayReusedAPIKeyTableName            = "api_keys"
	RelayReusedChannelTableName           = "channels"
	RelayReusedRequestTableName           = "requests"
	RelayReusedRequestExecutionTableName  = "request_executions"
	RelayReusedUsageLogTableName          = "usage_logs"
	RelayReusedProviderQuotaStatusTable   = "provider_quota_status"
	RelayReusedChannelModelPriceTableName = "channel_model_prices"
)

type RelayProductDataEntity struct {
	Table       string `json:"table"`
	Description string `json:"description"`
}

type RelayProductDataFoundationContract struct {
	Implemented []RelayProductDataEntity `json:"implemented"`
	Reused      []string                 `json:"reused"`
	Deferred    []RelayProductDataEntity `json:"deferred"`
}

type RelayProductCreateInput struct {
	Code                  string                   `json:"code"`
	Name                  string                   `json:"name"`
	ProviderType          RelayProductProviderType `json:"providerType"`
	AccessMode            RelayProductAccessMode   `json:"accessMode"`
	BillingMode           RelayProductBillingMode  `json:"billingMode"`
	Status                RelayProductStatus       `json:"status"`
	Currency              string                   `json:"currency"`
	AllowedModels         []string                 `json:"allowedModels"`
	RequestTimeoutSeconds int                      `json:"requestTimeoutSeconds"`
	ListPriceConfig       map[string]any           `json:"listPriceConfig"`
}

type RelayProductUpdateInput struct {
	Name                  *string                  `json:"name,omitempty"`
	BillingMode           *RelayProductBillingMode `json:"billingMode,omitempty"`
	Status                *RelayProductStatus      `json:"status,omitempty"`
	Currency              *string                  `json:"currency,omitempty"`
	AllowedModels         []string                 `json:"allowedModels,omitempty"`
	RequestTimeoutSeconds *int                     `json:"requestTimeoutSeconds,omitempty"`
	ListPriceConfig       map[string]any           `json:"listPriceConfig,omitempty"`
}

type RelayProductChannelBindingInput struct {
	ProductID     int                       `json:"productId"`
	ChannelID     int                       `json:"channelId"`
	Priority      int                       `json:"priority"`
	Weight        int                       `json:"weight"`
	Status        RelayProductChannelStatus `json:"status"`
	AllowFallback *bool                     `json:"allowFallback,omitempty"`
	ModelFilter   map[string]any            `json:"modelFilter,omitempty"`
	MaxInflight   *int                      `json:"maxInflight,omitempty"`
}

type RelayProductChannelBindingUpdateInput struct {
	Priority      *int                       `json:"priority,omitempty"`
	Weight        *int                       `json:"weight,omitempty"`
	Status        *RelayProductChannelStatus `json:"status,omitempty"`
	AllowFallback *bool                      `json:"allowFallback,omitempty"`
	ModelFilter   map[string]any             `json:"modelFilter,omitempty"`
	MaxInflight   *int                       `json:"maxInflight,omitempty"`
}

type RelayProductListInput struct {
	StatusIn     []RelayProductStatus      `json:"statusIn,omitempty"`
	ProviderType *RelayProductProviderType `json:"providerType,omitempty"`
	Query        string                    `json:"query,omitempty"`
}

type RelayProductRecord struct {
	ID                    int                      `json:"id"`
	Code                  string                   `json:"code"`
	Name                  string                   `json:"name"`
	ProviderType          RelayProductProviderType `json:"providerType"`
	AccessMode            RelayProductAccessMode   `json:"accessMode"`
	BillingMode           RelayProductBillingMode  `json:"billingMode"`
	Status                RelayProductStatus       `json:"status"`
	Currency              string                   `json:"currency"`
	AllowedModels         []string                 `json:"allowedModels"`
	RequestTimeoutSeconds int                      `json:"requestTimeoutSeconds"`
	ListPriceConfig       map[string]any           `json:"listPriceConfig"`
}

type RelayProductChannelBindingRecord struct {
	ID            int                       `json:"id"`
	ProductID     int                       `json:"productId"`
	ChannelID     int                       `json:"channelId"`
	Priority      int                       `json:"priority"`
	Weight        int                       `json:"weight"`
	Status        RelayProductChannelStatus `json:"status"`
	AllowFallback bool                      `json:"allowFallback"`
	ModelFilter   map[string]any            `json:"modelFilter"`
	MaxInflight   *int                      `json:"maxInflight,omitempty"`
}

var ErrRelayProductCodegenRequired = fmt.Errorf("relay product ent codegen is required before relay catalog CRUD can be enabled")

func NewRelayProductService(params RelayProductServiceParams) *RelayProductService {
	return &RelayProductService{
		AbstractService: &AbstractService{db: params.Ent},
		channelService:  params.ChannelService,
	}
}

func (s *RelayProductService) Contract() RelayProductContract {
	return RelayProductContract{
		ProviderTypes: []RelayProductProviderType{
			RelayProductProviderTypeClaudeCode,
			RelayProductProviderTypeCodex,
			RelayProductProviderTypeOpenAICompatible,
		},
		AccessModes: []RelayProductAccessMode{
			RelayProductAccessModeSharedCapacity,
		},
		BillingModes: []RelayProductBillingMode{
			RelayProductBillingModePrepaid,
			RelayProductBillingModeQuotaOnly,
		},
		Statuses: []RelayProductStatus{
			RelayProductStatusDraft,
			RelayProductStatusActive,
			RelayProductStatusArchived,
		},
		BindingStatus: []RelayProductChannelStatus{
			RelayProductChannelStatusActive,
			RelayProductChannelStatusPaused,
		},
		Defaults: RelayProductContractDefaults{
			Currency:              RelayProductDefaultCurrency,
			AccessMode:            string(RelayProductAccessModeSharedCapacity),
			BillingMode:           string(RelayProductBillingModePrepaid),
			Status:                string(RelayProductStatusDraft),
			RequestTimeoutSeconds: RelayProductDefaultRequestTimeoutSeconds,
			BindingWeight:         RelayProductChannelDefaultWeight,
		},
	}
}

func (s *RelayProductService) DataFoundationContract() RelayProductDataFoundationContract {
	return RelayProductDataFoundationContract{
		Implemented: []RelayProductDataEntity{
			{Table: RelayProductTableName, Description: "Sellable shared-capacity relay product catalog"},
			{Table: RelayProductChannelTableName, Description: "Product-to-upstream-channel pool bindings"},
		},
		Reused: []string{
			RelayReusedAPIKeyTableName,
			RelayReusedChannelTableName,
			RelayReusedRequestTableName,
			RelayReusedRequestExecutionTableName,
			RelayReusedUsageLogTableName,
			RelayReusedProviderQuotaStatusTable,
			RelayReusedChannelModelPriceTableName,
		},
		Deferred: []RelayProductDataEntity{
			{Table: RelayKeyTableName, Description: "Sub-Key business state bound to existing api_keys"},
			{Table: RelayWalletTableName, Description: "Fast balance snapshot for synchronous access checks"},
			{Table: RelayWalletLedgerEntryTableName, Description: "Immutable recharge, consume, refund, freeze and manual-adjust ledger"},
			{Table: RelayDailyUsageSummaryTableName, Description: "Daily aggregate for hard limits and operator dashboards"},
		},
	}
}

func (s *RelayProductService) ListRelayProducts(_ context.Context, input RelayProductListInput) ([]*RelayProductRecord, error) {
	if err := s.ValidateListRelayProductsInput(input); err != nil {
		return nil, err
	}
	return nil, ErrRelayProductCodegenRequired
}

func (s *RelayProductService) CreateRelayProduct(_ context.Context, input RelayProductCreateInput) (*RelayProductRecord, error) {
	if err := s.ValidateCreateRelayProductInput(input); err != nil {
		return nil, err
	}
	return nil, ErrRelayProductCodegenRequired
}

func (s *RelayProductService) UpdateRelayProduct(_ context.Context, id int, input RelayProductUpdateInput) (*RelayProductRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductInput(input); err != nil {
		return nil, err
	}
	return nil, ErrRelayProductCodegenRequired
}

func (s *RelayProductService) CreateRelayProductChannelBinding(ctx context.Context, input RelayProductChannelBindingInput) (*RelayProductChannelBindingRecord, error) {
	if err := s.ValidateCreateRelayProductChannelBinding(ctx, input); err != nil {
		return nil, err
	}
	return nil, ErrRelayProductCodegenRequired
}

func (s *RelayProductService) UpdateRelayProductChannelBinding(_ context.Context, id int, input RelayProductChannelBindingUpdateInput) (*RelayProductChannelBindingRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductChannelBinding(input); err != nil {
		return nil, err
	}
	return nil, ErrRelayProductCodegenRequired
}

func (s *RelayProductService) DeleteRelayProductChannelBinding(_ context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	return ErrRelayProductCodegenRequired
}

func (s *RelayProductService) ValidateListRelayProductsInput(input RelayProductListInput) error {
	if input.ProviderType != nil {
		if err := validateRelayProductProviderType(*input.ProviderType); err != nil {
			return err
		}
	}
	for _, status := range input.StatusIn {
		if err := validateRelayProductStatus(status); err != nil {
			return err
		}
	}
	return nil
}

func (s *RelayProductService) ValidateCreateRelayProductInput(input RelayProductCreateInput) error {
	if strings.TrimSpace(input.Code) == "" {
		return fmt.Errorf("relay product code is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("relay product name is required")
	}
	if err := validateRelayProductProviderType(input.ProviderType); err != nil {
		return err
	}
	if input.AccessMode != "" && input.AccessMode != RelayProductAccessModeSharedCapacity {
		return fmt.Errorf("unsupported relay product access mode %q", input.AccessMode)
	}
	if input.BillingMode != "" {
		if err := validateRelayProductBillingMode(input.BillingMode); err != nil {
			return err
		}
	}
	if input.Status != "" {
		if err := validateRelayProductStatus(input.Status); err != nil {
			return err
		}
	}
	if err := validateRelayProductCurrency(input.Currency); err != nil {
		return err
	}
	if err := validateRelayProductAllowedModels(input.AllowedModels); err != nil {
		return err
	}
	if err := validateRelayProductRequestTimeout(input.RequestTimeoutSeconds); err != nil {
		return err
	}
	return nil
}

func (s *RelayProductService) ValidateUpdateRelayProductInput(input RelayProductUpdateInput) error {
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return fmt.Errorf("relay product name cannot be empty")
	}
	if input.BillingMode != nil {
		if err := validateRelayProductBillingMode(*input.BillingMode); err != nil {
			return err
		}
	}
	if input.Status != nil {
		if err := validateRelayProductStatus(*input.Status); err != nil {
			return err
		}
	}
	if input.Currency != nil {
		if err := validateRelayProductCurrency(*input.Currency); err != nil {
			return err
		}
	}
	if input.AllowedModels != nil {
		if err := validateRelayProductAllowedModels(input.AllowedModels); err != nil {
			return err
		}
	}
	if input.RequestTimeoutSeconds != nil {
		if err := validateRelayProductRequestTimeout(*input.RequestTimeoutSeconds); err != nil {
			return err
		}
	}
	return nil
}

func (s *RelayProductService) ValidateCreateRelayProductChannelBinding(ctx context.Context, input RelayProductChannelBindingInput) error {
	if input.ProductID <= 0 {
		return fmt.Errorf("relay product id must be greater than 0")
	}
	if err := s.validateRelayChannel(ctx, input.ChannelID); err != nil {
		return err
	}
	if input.Weight <= 0 {
		return fmt.Errorf("relay product channel weight must be greater than 0")
	}
	if err := validateRelayProductChannelStatus(input.Status); err != nil {
		return err
	}
	if input.MaxInflight != nil && *input.MaxInflight <= 0 {
		return fmt.Errorf("relay product channel max inflight must be greater than 0 when set")
	}
	return nil
}

func (s *RelayProductService) ValidateUpdateRelayProductChannelBinding(input RelayProductChannelBindingUpdateInput) error {
	if input.Weight != nil && *input.Weight <= 0 {
		return fmt.Errorf("relay product channel weight must be greater than 0")
	}
	if input.Status != nil {
		if err := validateRelayProductChannelStatus(*input.Status); err != nil {
			return err
		}
	}
	if input.MaxInflight != nil && *input.MaxInflight <= 0 {
		return fmt.Errorf("relay product channel max inflight must be greater than 0 when set")
	}
	return nil
}

func (s *RelayProductService) ValidateRelayBindingChannel(ctx context.Context, channelID int) (*ent.Channel, error) {
	if err := s.validateRelayChannel(ctx, channelID); err != nil {
		return nil, err
	}
	return s.entFromContext(ctx).Channel.Get(ctx, channelID)
}

func (s *RelayProductService) validateRelayChannel(ctx context.Context, channelID int) error {
	if channelID <= 0 {
		return fmt.Errorf("relay product channel id must be greater than 0")
	}
	ch, err := s.entFromContext(ctx).Channel.Get(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to load relay product channel %d: %w", channelID, err)
	}
	if ch.Status == channel.StatusArchived {
		return fmt.Errorf("channel %d is archived and cannot be added to a relay product pool", channelID)
	}
	return nil
}

func validateRelayProductProviderType(value RelayProductProviderType) error {
	allowed := []RelayProductProviderType{
		RelayProductProviderTypeClaudeCode,
		RelayProductProviderTypeCodex,
		RelayProductProviderTypeOpenAICompatible,
	}
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("unsupported relay product provider type %q", value)
	}
	return nil
}

func validateRelayProductBillingMode(value RelayProductBillingMode) error {
	allowed := []RelayProductBillingMode{
		RelayProductBillingModePrepaid,
		RelayProductBillingModeQuotaOnly,
	}
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("unsupported relay product billing mode %q", value)
	}
	return nil
}

func validateRelayProductStatus(value RelayProductStatus) error {
	allowed := []RelayProductStatus{
		RelayProductStatusDraft,
		RelayProductStatusActive,
		RelayProductStatusArchived,
	}
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("unsupported relay product status %q", value)
	}
	return nil
}

func validateRelayProductChannelStatus(value RelayProductChannelStatus) error {
	allowed := []RelayProductChannelStatus{
		RelayProductChannelStatusActive,
		RelayProductChannelStatusPaused,
	}
	if !slices.Contains(allowed, value) {
		return fmt.Errorf("unsupported relay product channel status %q", value)
	}
	return nil
}

func validateRelayProductCurrency(value string) error {
	if value == "" {
		return nil
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("relay product currency cannot be blank")
	}
	return nil
}

func validateRelayProductAllowedModels(models []string) error {
	for _, modelID := range models {
		if strings.TrimSpace(modelID) == "" {
			return fmt.Errorf("relay product allowed models cannot include empty values")
		}
	}
	return nil
}

func validateRelayProductRequestTimeout(seconds int) error {
	if seconds == 0 {
		return nil
	}
	if seconds < 0 {
		return fmt.Errorf("relay product request timeout must be greater than or equal to 0")
	}
	return nil
}
