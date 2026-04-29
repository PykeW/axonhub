package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"entgo.io/ent/dialect"
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

func (s *RelayProductService) ListRelayProducts(ctx context.Context, input RelayProductListInput) ([]*RelayProductRecord, error) {
	if err := s.ValidateListRelayProductsInput(input); err != nil {
		return nil, err
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}

	clauses := []string{"deleted_at = 0"}
	args := make([]any, 0, len(input.StatusIn)+2)
	if len(input.StatusIn) > 0 {
		placeholders := make([]string, len(input.StatusIn))
		for i, status := range input.StatusIn {
			args = append(args, string(status))
			placeholders[i] = relayPlaceholder(dialectName, len(args))
		}
		clauses = append(clauses, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}
	if input.ProviderType != nil {
		args = append(args, string(*input.ProviderType))
		clauses = append(clauses, fmt.Sprintf("provider_type = %s", relayPlaceholder(dialectName, len(args))))
	}
	if query := strings.TrimSpace(input.Query); query != "" {
		pattern := "%" + strings.ToLower(query) + "%"
		args = append(args, pattern)
		codePlaceholder := relayPlaceholder(dialectName, len(args))
		args = append(args, pattern)
		namePlaceholder := relayPlaceholder(dialectName, len(args))
		clauses = append(clauses, fmt.Sprintf("(LOWER(code) LIKE %s OR LOWER(name) LIKE %s)", codePlaceholder, namePlaceholder))
	}

	query := fmt.Sprintf(`SELECT id, code, name, provider_type, access_mode, billing_mode, status, currency, allowed_models, request_timeout_seconds, list_price_config
FROM relay_products
WHERE %s
ORDER BY id ASC`, strings.Join(clauses, " AND "))
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list relay products: %w", err)
	}
	defer rows.Close()

	products := make([]*RelayProductRecord, 0)
	for rows.Next() {
		product, err := scanRelayProductRecord(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate relay products: %w", err)
	}

	return products, nil
}

func (s *RelayProductService) CreateRelayProduct(ctx context.Context, input RelayProductCreateInput) (*RelayProductRecord, error) {
	if err := s.ValidateCreateRelayProductInput(input); err != nil {
		return nil, err
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}

	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
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

	allowedModels, err := marshalRelayJSON(input.AllowedModels)
	if err != nil {
		return nil, err
	}
	priceConfig, err := marshalRelayJSONObject(input.ListPriceConfig)
	if err != nil {
		return nil, err
	}

	columns := []string{"code", "name", "provider_type", "access_mode", "billing_mode", "status", "currency", "allowed_models", "request_timeout_seconds", "list_price_config"}
	placeholders := relayPlaceholders(dialectName, len(columns), 1)
	query := fmt.Sprintf("INSERT INTO relay_products (%s) VALUES (%s)", strings.Join(columns, ","), strings.Join(placeholders, ","))
	args := []any{code, name, string(input.ProviderType), string(accessMode), string(billingMode), string(status), currency, allowedModels, timeout, priceConfig}

	id, err := execRelayInsert(ctx, db, dialectName, query, args...)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("relay product code %q: %w: %v", code, ErrRelayProductCodeExists, err)
		}
		return nil, fmt.Errorf("failed to create relay product: %w", err)
	}

	return s.getRelayProduct(ctx, id)
}

func (s *RelayProductService) UpdateRelayProduct(ctx context.Context, id int, input RelayProductUpdateInput) (*RelayProductRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductInput(input); err != nil {
		return nil, err
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}

	sets := make([]string, 0, 8)
	args := make([]any, 0, 8)
	if input.Name != nil {
		args = append(args, strings.TrimSpace(*input.Name))
		sets = append(sets, fmt.Sprintf("name = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.BillingMode != nil {
		args = append(args, string(*input.BillingMode))
		sets = append(sets, fmt.Sprintf("billing_mode = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.Status != nil {
		args = append(args, string(*input.Status))
		sets = append(sets, fmt.Sprintf("status = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.Currency != nil {
		args = append(args, strings.TrimSpace(*input.Currency))
		sets = append(sets, fmt.Sprintf("currency = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.AllowedModels != nil {
		allowedModels, err := marshalRelayJSON(input.AllowedModels)
		if err != nil {
			return nil, err
		}
		args = append(args, allowedModels)
		sets = append(sets, fmt.Sprintf("allowed_models = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.RequestTimeoutSeconds != nil {
		args = append(args, *input.RequestTimeoutSeconds)
		sets = append(sets, fmt.Sprintf("request_timeout_seconds = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.ListPriceConfig != nil {
		priceConfig, err := marshalRelayJSONObject(input.ListPriceConfig)
		if err != nil {
			return nil, err
		}
		args = append(args, priceConfig)
		sets = append(sets, fmt.Sprintf("list_price_config = %s", relayPlaceholder(dialectName, len(args))))
	}
	if len(sets) == 0 {
		return s.getRelayProduct(ctx, id)
	}
	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")

	args = append(args, id)
	idPlaceholder := relayPlaceholder(dialectName, len(args))
	query := fmt.Sprintf("UPDATE relay_products SET %s WHERE id = %s AND deleted_at = 0", strings.Join(sets, ", "), idPlaceholder)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update relay product %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect relay product update result: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
	}

	return s.getRelayProduct(ctx, id)
}

func (s *RelayProductService) CreateRelayProductChannelBinding(ctx context.Context, input RelayProductChannelBindingInput) (*RelayProductChannelBindingRecord, error) {
	if err := s.ValidateCreateRelayProductChannelBinding(ctx, input); err != nil {
		return nil, err
	}
	if _, err := s.getRelayProduct(ctx, input.ProductID); err != nil {
		return nil, err
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
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
	modelFilter, err := marshalRelayJSONObject(input.ModelFilter)
	if err != nil {
		return nil, err
	}

	columns := []string{"product_id", "channel_id", "priority", "weight", "status", "allow_fallback", "model_filter", "max_inflight"}
	placeholders := relayPlaceholders(dialectName, len(columns), 1)
	query := fmt.Sprintf("INSERT INTO relay_product_channels (%s) VALUES (%s)", strings.Join(columns, ","), strings.Join(placeholders, ","))
	args := []any{input.ProductID, input.ChannelID, input.Priority, input.Weight, string(status), allowFallback, modelFilter, nullableIntPtr(input.MaxInflight)}

	id, err := execRelayInsert(ctx, db, dialectName, query, args...)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("relay product %d is already bound to channel %d: %w: %v", input.ProductID, input.ChannelID, ErrRelayProductAlreadyBound, err)
		}
		return nil, fmt.Errorf("failed to create relay product channel binding: %w", err)
	}

	return s.getRelayProductChannelBinding(ctx, id)
}

func (s *RelayProductService) UpdateRelayProductChannelBinding(ctx context.Context, id int, input RelayProductChannelBindingUpdateInput) (*RelayProductChannelBindingRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product channel binding id must be greater than 0")
	}
	if err := s.ValidateUpdateRelayProductChannelBinding(input); err != nil {
		return nil, err
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}

	sets := make([]string, 0, 7)
	args := make([]any, 0, 7)
	if input.Priority != nil {
		args = append(args, *input.Priority)
		sets = append(sets, fmt.Sprintf("priority = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.Weight != nil {
		args = append(args, *input.Weight)
		sets = append(sets, fmt.Sprintf("weight = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.Status != nil {
		args = append(args, string(*input.Status))
		sets = append(sets, fmt.Sprintf("status = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.AllowFallback != nil {
		args = append(args, *input.AllowFallback)
		sets = append(sets, fmt.Sprintf("allow_fallback = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.ModelFilter != nil {
		modelFilter, err := marshalRelayJSONObject(input.ModelFilter)
		if err != nil {
			return nil, err
		}
		args = append(args, modelFilter)
		sets = append(sets, fmt.Sprintf("model_filter = %s", relayPlaceholder(dialectName, len(args))))
	}
	if input.MaxInflight != nil {
		args = append(args, *input.MaxInflight)
		sets = append(sets, fmt.Sprintf("max_inflight = %s", relayPlaceholder(dialectName, len(args))))
	}
	if len(sets) == 0 {
		return s.getRelayProductChannelBinding(ctx, id)
	}
	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")

	args = append(args, id)
	idPlaceholder := relayPlaceholder(dialectName, len(args))
	query := fmt.Sprintf("UPDATE relay_product_channels SET %s WHERE id = %s", strings.Join(sets, ", "), idPlaceholder)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update relay product channel binding %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect relay product channel binding update result: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
	}

	return s.getRelayProductChannelBinding(ctx, id)
}

func (s *RelayProductService) DeleteRelayProductChannelBinding(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("relay product channel binding id must be greater than 0")
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM relay_product_channels WHERE id = %s", relayPlaceholder(dialectName, 1))
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete relay product channel binding %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect relay product channel binding delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
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

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT id, code, name, provider_type, access_mode, billing_mode, status, currency, allowed_models, request_timeout_seconds, list_price_config
FROM relay_products
WHERE id = %s AND deleted_at = 0
LIMIT 1`, relayPlaceholder(dialectName, 1))
	product, err := scanRelayProductRecord(db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("relay product %d: %w", id, ErrRelayProductNotFound)
		}
		return nil, fmt.Errorf("failed to load relay product %d: %w", id, err)
	}

	return product, nil
}

func (s *RelayProductService) getRelayProductChannelBinding(ctx context.Context, id int) (*RelayProductChannelBindingRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("relay product channel binding id must be greater than 0")
	}

	db, dialectName, err := relaySQLDB(s.entFromContext(ctx))
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT id, product_id, channel_id, priority, weight, status, allow_fallback, model_filter, max_inflight
FROM relay_product_channels
WHERE id = %s
LIMIT 1`, relayPlaceholder(dialectName, 1))
	binding, err := scanRelayProductChannelBindingRecord(db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("relay product channel binding %d: %w", id, ErrRelayBindingNotFound)
		}
		return nil, fmt.Errorf("failed to load relay product channel binding %d: %w", id, err)
	}

	return binding, nil
}

func scanRelayProductRecord(row relayScanner) (*RelayProductRecord, error) {
	var (
		product         RelayProductRecord
		providerType    string
		accessMode      string
		billingMode     string
		status          string
		allowedModels   sql.NullString
		listPriceConfig sql.NullString
	)
	if err := row.Scan(
		&product.ID,
		&product.Code,
		&product.Name,
		&providerType,
		&accessMode,
		&billingMode,
		&status,
		&product.Currency,
		&allowedModels,
		&product.RequestTimeoutSeconds,
		&listPriceConfig,
	); err != nil {
		return nil, err
	}

	models, err := parseRelayStringList(allowedModels)
	if err != nil {
		return nil, err
	}
	priceConfig, err := parseRelayJSONObject(listPriceConfig)
	if err != nil {
		return nil, err
	}

	product.ProviderType = RelayProductProviderType(providerType)
	product.AccessMode = RelayProductAccessMode(accessMode)
	product.BillingMode = RelayProductBillingMode(billingMode)
	product.Status = RelayProductStatus(status)
	product.AllowedModels = models
	product.ListPriceConfig = priceConfig
	if product.AllowedModels == nil {
		product.AllowedModels = []string{}
	}

	return &product, nil
}

func scanRelayProductChannelBindingRecord(row relayScanner) (*RelayProductChannelBindingRecord, error) {
	var (
		binding     RelayProductChannelBindingRecord
		status      string
		modelFilter sql.NullString
		maxInflight sql.NullInt64
	)
	if err := row.Scan(
		&binding.ID,
		&binding.ProductID,
		&binding.ChannelID,
		&binding.Priority,
		&binding.Weight,
		&status,
		&binding.AllowFallback,
		&modelFilter,
		&maxInflight,
	); err != nil {
		return nil, err
	}

	parsedFilter, err := parseRelayJSONObject(modelFilter)
	if err != nil {
		return nil, err
	}
	if maxInflight.Valid {
		value := int(maxInflight.Int64)
		binding.MaxInflight = &value
	}
	binding.Status = RelayProductChannelStatus(status)
	binding.ModelFilter = parsedFilter

	return &binding, nil
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

func execRelayInsert(ctx context.Context, db *sql.DB, dialectName, query string, args ...any) (int, error) {
	if dialectName == dialect.Postgres {
		var id int
		if err := db.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func relayPlaceholders(dialectName string, count, start int) []string {
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = relayPlaceholder(dialectName, start+i)
	}

	return placeholders
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}
