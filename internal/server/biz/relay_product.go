package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/relayproduct"
	"github.com/looplj/axonhub/internal/ent/relayproductchannel"
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

func NewRelayProductService(params RelayProductServiceParams) *RelayProductService {
	return &RelayProductService{
		AbstractService: &AbstractService{db: params.Ent},
		channelService:  params.ChannelService,
	}
}

func relayProductFromEnt(e *ent.RelayProduct) *RelayProductRecord {
	models := e.AllowedModels
	if models == nil {
		models = []string{}
	}
	return &RelayProductRecord{
		ID:                    e.ID,
		Code:                  e.Code,
		Name:                  e.Name,
		ProviderType:          RelayProductProviderType(e.ProviderType),
		AccessMode:            RelayProductAccessMode(e.AccessMode),
		BillingMode:           RelayProductBillingMode(e.BillingMode),
		Status:                RelayProductStatus(e.Status),
		Currency:              e.Currency,
		AllowedModels:         models,
		RequestTimeoutSeconds: e.RequestTimeoutSeconds,
		ListPriceConfig:       e.ListPriceConfig,
	}
}

func relayProductChannelFromEnt(e *ent.RelayProductChannel) *RelayProductChannelBindingRecord {
	binding := &RelayProductChannelBindingRecord{
		ID:            e.ID,
		ProductID:     e.ProductID,
		ChannelID:     e.ChannelID,
		Priority:      e.Priority,
		Weight:        e.Weight,
		Status:        RelayProductChannelStatus(e.Status),
		AllowFallback: e.AllowFallback,
		ModelFilter:   func() map[string]any {
			if mf, ok := e.ModelFilter.(map[string]any); ok {
				return mf
			}
			return map[string]any{}
		}(),
	}
	if e.MaxInflight != nil {
		v := *e.MaxInflight
		binding.MaxInflight = &v
	}
	return binding
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

func (s *RelayProductService) ListRelayProducts(ctx context.Context, input RelayProductListInput) ([]*RelayProductRecord, error) {
	if err := s.ValidateListRelayProductsInput(input); err != nil {
		return nil, err
	}

	q := s.entFromContext(ctx).RelayProduct.Query().Where(relayproduct.DeletedAt(0))
	if len(input.StatusIn) > 0 {
		statuses := make([]relayproduct.Status, len(input.StatusIn))
		for i, status := range input.StatusIn {
			statuses[i] = relayproduct.Status(status)
		}
		q = q.Where(relayproduct.StatusIn(statuses...))
	}
	if input.ProviderType != nil {
		q = q.Where(relayproduct.ProviderTypeEQ(relayproduct.ProviderType(*input.ProviderType)))
	}
	if query := strings.TrimSpace(input.Query); query != "" {
		q = q.Where(relayproduct.Or(
			relayproduct.CodeContains(query),
			relayproduct.NameContains(query),
		))
	}

	ents, err := q.Order(ent.Asc(relayproduct.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list relay products: %w", err)
	}
	records := make([]*RelayProductRecord, len(ents))
	for i, e := range ents {
		records[i] = relayProductFromEnt(e)
	}
	return records, nil
}

func (s *RelayProductService) CreateRelayProduct(ctx context.Context, input RelayProductCreateInput) (*RelayProductRecord, error) {
	if err := s.ValidateCreateRelayProductInput(input); err != nil {
		return nil, err
	}

	accessMode := input.AccessMode
	if accessMode == "" {
		accessMode = RelayProductAccessModeSharedCapacity
	}
	billingMode := input.BillingMode
	if billingMode == "" {
		billingMode = RelayProductBillingModePrepaid
	}
	status := input.Status
	if status == "" {
		status = RelayProductStatusDraft
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = RelayProductDefaultCurrency
	}
	timeout := input.RequestTimeoutSeconds
	if timeout == 0 {
		timeout = RelayProductDefaultRequestTimeoutSeconds
	}
	allowedModels := input.AllowedModels
	if allowedModels == nil {
		allowedModels = []string{}
	}
	priceConfig := input.ListPriceConfig
	if priceConfig == nil {
		priceConfig = map[string]any{}
	}

	e, err := s.entFromContext(ctx).RelayProduct.Create().
		SetCode(strings.TrimSpace(input.Code)).
		SetName(strings.TrimSpace(input.Name)).
		SetProviderType(relayproduct.ProviderType(input.ProviderType)).
		SetAccessMode(relayproduct.AccessMode(accessMode)).
		SetBillingMode(relayproduct.BillingMode(billingMode)).
		SetStatus(relayproduct.Status(status)).
		SetCurrency(currency).
		SetAllowedModels(allowedModels).
		SetRequestTimeoutSeconds(timeout).
		SetListPriceConfig(priceConfig).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("relay product code %q: %w: %v", input.Code, ErrRelayProductCodeExists, err)
		}
		return nil, fmt.Errorf("failed to create relay product: %w", err)
	}
	return relayProductFromEnt(e), nil
}

func (s *RelayProductService) UpdateRelayProduct(ctx context.Context, id int, input RelayProductUpdateInput) (*RelayProductRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductInput(input); err != nil {
		return nil, err
	}

	u := s.entFromContext(ctx).RelayProduct.UpdateOneID(id).Where(relayproduct.DeletedAt(0))
	if input.Name != nil {
		u = u.SetName(strings.TrimSpace(*input.Name))
	}
	if input.BillingMode != nil {
		u = u.SetBillingMode(relayproduct.BillingMode(*input.BillingMode))
	}
	if input.Status != nil {
		u = u.SetStatus(relayproduct.Status(*input.Status))
	}
	if input.Currency != nil {
		u = u.SetCurrency(strings.TrimSpace(*input.Currency))
	}
	if input.AllowedModels != nil {
		u = u.SetAllowedModels(input.AllowedModels)
	}
	if input.RequestTimeoutSeconds != nil {
		u = u.SetRequestTimeoutSeconds(*input.RequestTimeoutSeconds)
	}
	if input.ListPriceConfig != nil {
		u = u.SetListPriceConfig(input.ListPriceConfig)
	}
	e, err := u.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
		}
		return nil, fmt.Errorf("failed to update relay product %d: %w", id, err)
	}
	return relayProductFromEnt(e), nil
}

func (s *RelayProductService) CreateRelayProductChannelBinding(ctx context.Context, input RelayProductChannelBindingInput) (*RelayProductChannelBindingRecord, error) {
	if err := s.ValidateCreateRelayProductChannelBinding(ctx, input); err != nil {
		return nil, err
	}
	if _, err := s.getRelayProduct(ctx, input.ProductID); err != nil {
		return nil, err
	}

	status := input.Status
	if status == "" {
		status = RelayProductChannelStatusActive
	}
	allowFallback := true
	if input.AllowFallback != nil {
		allowFallback = *input.AllowFallback
	}
	modelFilter := input.ModelFilter
	if modelFilter == nil {
		modelFilter = map[string]any{}
	}

	b := s.entFromContext(ctx).RelayProductChannel.Create().
		SetProductID(input.ProductID).
		SetChannelID(input.ChannelID).
		SetPriority(input.Priority).
		SetWeight(input.Weight).
		SetStatus(relayproductchannel.Status(status)).
		SetAllowFallback(allowFallback).
		SetModelFilter(modelFilter)
	if input.MaxInflight != nil {
		b = b.SetMaxInflight(*input.MaxInflight)
	}
	e, err := b.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("relay product %d is already bound to channel %d: %w: %v", input.ProductID, input.ChannelID, ErrRelayProductAlreadyBound, err)
		}
		return nil, fmt.Errorf("failed to create relay product channel binding: %w", err)
	}
	return relayProductChannelFromEnt(e), nil
}

func (s *RelayProductService) UpdateRelayProductChannelBinding(ctx context.Context, id int, input RelayProductChannelBindingUpdateInput) (*RelayProductChannelBindingRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductChannelBinding(input); err != nil {
		return nil, err
	}

	u := s.entFromContext(ctx).RelayProductChannel.UpdateOneID(id)
	if input.Priority != nil {
		u = u.SetPriority(*input.Priority)
	}
	if input.Weight != nil {
		u = u.SetWeight(*input.Weight)
	}
	if input.Status != nil {
		u = u.SetStatus(relayproductchannel.Status(*input.Status))
	}
	if input.AllowFallback != nil {
		u = u.SetAllowFallback(*input.AllowFallback)
	}
	if input.ModelFilter != nil {
		u = u.SetModelFilter(input.ModelFilter)
	}
	if input.MaxInflight != nil {
		u = u.SetMaxInflight(*input.MaxInflight)
	}
	e, err := u.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
		}
		return nil, fmt.Errorf("failed to update relay product channel binding %d: %w", id, err)
	}
	return relayProductChannelFromEnt(e), nil
}

func (s *RelayProductService) DeleteRelayProductChannelBinding(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	err := s.entFromContext(ctx).RelayProductChannel.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
		}
		return fmt.Errorf("failed to delete relay product channel binding %d: %w", id, err)
	}
	return nil
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

func (s *RelayProductService) getRelayProduct(ctx context.Context, id int) (*RelayProductRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	e, err := s.entFromContext(ctx).RelayProduct.Query().
		Where(relayproduct.ID(id), relayproduct.DeletedAt(0)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
		}
		return nil, fmt.Errorf("failed to load relay product %d: %w", id, err)
	}
	return relayProductFromEnt(e), nil
}

func (s *RelayProductService) getRelayProductChannelBinding(ctx context.Context, id int) (*RelayProductChannelBindingRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	e, err := s.entFromContext(ctx).RelayProductChannel.Query().
		Where(relayproductchannel.ID(id)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
		}
		return nil, fmt.Errorf("failed to load relay product channel binding %d: %w", id, err)
	}
	return relayProductChannelFromEnt(e), nil
}

func marshalRelayJSON(value any) (string, error) {
	if value == nil {
		value = []string{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode relay JSON value: %w", err)
	}

	return string(payload), nil
}

func marshalRelayJSONObject(value map[string]any) (string, error) {
	if value == nil {
		value = map[string]any{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode relay JSON object: %w", err)
	}

	return string(payload), nil
}

func parseRelayJSONObject(value sql.NullString) (map[string]any, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(value.String), &out); err != nil {
		return nil, fmt.Errorf("invalid relay JSON object %q: %w", value.String, err)
	}
	if out == nil {
		return map[string]any{}, nil
	}

	return out, nil
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}
