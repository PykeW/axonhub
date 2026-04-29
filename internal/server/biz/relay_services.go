package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/ent"
)

type RelayAccessContext = RelayAuthContext

type RelayAccessServiceParams struct {
	fx.In

	Ent    *ent.Client
	Router *RelayRouterService
}

type RelayAccessService struct {
	*AbstractService

	router *RelayRouterService
}

func NewRelayAccessService(params RelayAccessServiceParams) *RelayAccessService {
	return &RelayAccessService{
		AbstractService: &AbstractService{db: params.Ent},
		router:          params.Router,
	}
}

func (s *RelayAccessService) ResolveRelayAuth(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
	if apiKey == nil {
		return nil, nil
	}

	return s.LoadContextByAPIKey(ctx, apiKey.ID)
}

func (s *RelayAccessService) LoadContextByAPIKey(ctx context.Context, apiKeyID int) (*RelayAccessContext, error) {
	if apiKeyID <= 0 {
		return nil, fmt.Errorf("relay api key id must be greater than 0")
	}

	db, dialectName, err := relaySQLDB(s.db)
	if err != nil {
		return nil, err
	}

	statDate := relayUTCDate(time.Now())
	query := fmt.Sprintf(`
SELECT
	rk.id,
	rk.api_key_id,
	rk.project_id,
	rk.product_id,
	rk.status,
	rk.balance_mode,
	rk.daily_request_limit,
	rk.daily_token_limit,
	rk.monthly_cost_limit,
	rk.concurrency_limit,
	rk.expires_at,
	rp.code,
	rp.name,
	rp.provider_type,
	rp.status,
	rw.currency,
	rw.available_amount,
	rw.frozen_amount,
	rw.overdraft_limit,
	rw.version,
	rdus.request_count,
	rdus.total_tokens,
	rdus.total_charge
FROM relay_keys rk
JOIN api_keys ak ON ak.id = rk.api_key_id AND ak.project_id = rk.project_id
JOIN relay_products rp ON rp.id = rk.product_id AND rp.deleted_at = 0
LEFT JOIN relay_wallets rw ON rw.relay_key_id = rk.id
LEFT JOIN relay_daily_usage_summaries rdus ON rdus.relay_key_id = rk.id AND rdus.stat_date = %s
WHERE rk.api_key_id = %s AND rk.deleted_at = 0
LIMIT 1`, relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2))

	row := db.QueryRowContext(ctx, query, statDate, apiKeyID)
	relay, err := scanRelayAccessContext(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if isMissingRelayTableError(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to load relay access context: %w", err)
	}

	if s.router != nil {
		pool, err := s.router.listActivePool(ctx, relay.ProductID, "")
		if err != nil {
			return nil, err
		}
		relay.ChannelPool = pool
	}

	return relay, nil
}

func (s *RelayAccessService) CheckAccess(ctx context.Context, relay *RelayAccessContext, input RelayAccessCheckInput) error {
	decision, err := s.CheckRelayAccess(ctx, relay, input)
	if err != nil {
		return err
	}

	return decision.ErrorOrNil()
}

func (s *RelayAccessService) CheckRelayAccess(_ context.Context, relay *RelayAuthContext, input RelayAccessCheckInput) (*RelayAccessDecision, error) {
	if relay == nil {
		return allowRelayAccess(), nil
	}

	if input.APIKey != nil {
		if input.APIKey.ID != 0 && relay.APIKeyID != 0 && input.APIKey.ID != relay.APIKeyID {
			return denyRelayAccess(http.StatusForbidden, "relay_api_key_mismatch", "relay key is not bound to this API key"), nil
		}
		if input.APIKey.ProjectID != 0 && relay.ProjectID != 0 && input.APIKey.ProjectID != relay.ProjectID {
			return denyRelayAccess(http.StatusForbidden, "relay_project_scope_mismatch", "relay key is outside the API key project"), nil
		}
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	if relay.ProductStatus != "" && relay.ProductStatus != RelayProductStatusActive {
		return denyRelayAccess(http.StatusForbidden, "relay_product_inactive", fmt.Sprintf("relay product is %s", relay.ProductStatus)), nil
	}
	if relay.Status != "" && relay.Status != RelayKeyStatusActive {
		return denyRelayAccess(http.StatusForbidden, "relay_key_inactive", fmt.Sprintf("relay key is %s", relay.Status)), nil
	}
	if relay.ExpiresAt != nil && !relay.ExpiresAt.After(now) {
		return denyRelayAccess(http.StatusForbidden, "relay_key_expired", "relay key has expired"), nil
	}
	if relay.BalanceMode == RelayKeyBalanceModePrepaid {
		if relay.Wallet == nil {
			return denyRelayAccess(http.StatusPaymentRequired, "relay_wallet_missing", "relay key wallet is missing"), nil
		}
		available := relay.Wallet.AvailableAmount.Add(relay.Wallet.OverdraftLimit)
		if available.LessThanOrEqual(decimal.Zero) {
			return denyRelayAccess(http.StatusPaymentRequired, "relay_balance_exhausted", "relay key balance exhausted"), nil
		}
	}
	if relay.Quota.DailyRequestLimit != nil && relay.DailyUsage.RequestCount >= *relay.Quota.DailyRequestLimit {
		return denyRelayAccess(http.StatusForbidden, "relay_daily_request_quota_exceeded", "relay key daily request quota exceeded"), nil
	}
	if relay.Quota.DailyTokenLimit != nil && relay.DailyUsage.TotalTokens >= *relay.Quota.DailyTokenLimit {
		return denyRelayAccess(http.StatusForbidden, "relay_daily_token_quota_exceeded", "relay key daily token quota exceeded"), nil
	}

	return allowRelayAccess(), nil
}

type RelayRouterService struct {
	*AbstractService
}

func NewRelayRouterService(ent *ent.Client) *RelayRouterService {
	return &RelayRouterService{AbstractService: &AbstractService{db: ent}}
}

func (s *RelayRouterService) ListActiveChannelIDs(ctx context.Context, productID int, model string) ([]int, error) {
	pool, err := s.listActivePool(ctx, productID, model)
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(pool.Channels))
	for _, entry := range pool.Channels {
		ids = append(ids, entry.ChannelID)
	}

	return ids, nil
}

func (s *RelayRouterService) listActivePool(ctx context.Context, productID int, model string) (RelayChannelPool, error) {
	if productID <= 0 {
		return RelayChannelPool{}, fmt.Errorf("relay product id must be greater than 0")
	}

	db, dialectName, err := relaySQLDB(s.db)
	if err != nil {
		return RelayChannelPool{}, err
	}

	query := fmt.Sprintf(`
SELECT
	rp.allowed_models,
	rpc.channel_id,
	rpc.priority,
	rpc.weight,
	rpc.allow_fallback,
	rpc.model_filter,
	rpc.max_inflight
FROM relay_products rp
JOIN relay_product_channels rpc ON rpc.product_id = rp.id
JOIN channels c ON c.id = rpc.channel_id AND c.deleted_at = 0 AND c.status = 'enabled'
WHERE rp.id = %s
  AND rp.deleted_at = 0
  AND rp.status = 'active'
  AND rpc.status = 'active'
ORDER BY rpc.priority ASC, rpc.weight DESC, rpc.channel_id ASC`, relayPlaceholder(dialectName, 1))

	rows, err := db.QueryContext(ctx, query, productID)
	if err != nil {
		if isMissingRelayTableError(err) {
			return RelayChannelPool{}, nil
		}

		return RelayChannelPool{}, fmt.Errorf("failed to list relay product channels: %w", err)
	}
	defer rows.Close()

	pool := RelayChannelPool{}
	allowedModelsLoaded := false
	for rows.Next() {
		var (
			allowedModelsRaw sql.NullString
			modelFilterRaw   sql.NullString
			entry            RelayChannelPoolEntry
			maxInflight      sql.NullInt64
		)
		if err := rows.Scan(
			&allowedModelsRaw,
			&entry.ChannelID,
			&entry.Priority,
			&entry.Weight,
			&entry.AllowFallback,
			&modelFilterRaw,
			&maxInflight,
		); err != nil {
			return RelayChannelPool{}, fmt.Errorf("failed to scan relay product channel: %w", err)
		}
		if !allowedModelsLoaded {
			models, err := parseRelayStringList(allowedModelsRaw)
			if err != nil {
				return RelayChannelPool{}, err
			}
			pool.AllowedModels = models
			allowedModelsLoaded = true
		}
		if maxInflight.Valid {
			value := int(maxInflight.Int64)
			entry.MaxInflight = &value
		}

		entry.ModelFilter = parseRelayModelFilter(modelFilterRaw)
		if model != "" && (!matchRelayModelPatterns(pool.AllowedModels, model) || !entry.AllowsModel(model)) {
			continue
		}
		pool.Channels = append(pool.Channels, entry)
	}
	if err := rows.Err(); err != nil {
		return RelayChannelPool{}, fmt.Errorf("failed to iterate relay product channels: %w", err)
	}

	return pool, nil
}

type RelaySettlementServiceParams struct {
	fx.In

	Ent *ent.Client
}

type RelaySettlementService struct {
	*AbstractService
}

func NewRelaySettlementService(params RelaySettlementServiceParams) *RelaySettlementService {
	return &RelaySettlementService{AbstractService: &AbstractService{db: params.Ent}}
}

func (s *RelaySettlementService) RecordRelayUsage(ctx context.Context, relay *RelayAuthContext, input RelayUsageSettlementInput) error {
	return s.SettleUsage(ctx, relay, input)
}

func (s *RelaySettlementService) SettleUsage(ctx context.Context, relay *RelayAccessContext, input RelayUsageSettlementInput) error {
	if relay == nil || input.UsageLog == nil {
		return nil
	}
	if relay.RelayKeyID <= 0 {
		return fmt.Errorf("relay key id must be greater than 0")
	}
	if input.UsageLog.ID <= 0 {
		return fmt.Errorf("usage log id must be greater than 0 for relay settlement")
	}

	db, dialectName, err := relaySQLDB(s.db)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin relay settlement transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	idempotencyKey := fmt.Sprintf("usage_log:%d", input.UsageLog.ID)
	exists, err := relayLedgerEntryExists(ctx, tx, dialectName, idempotencyKey)
	if err != nil {
		return err
	}
	if exists {
		committed = true
		return tx.Commit()
	}

	charge := decimal.Zero
	if input.UsageLog.TotalCost != nil {
		charge = decimal.NewFromFloat(*input.UsageLog.TotalCost)
	}
	if charge.IsNegative() {
		return fmt.Errorf("relay settlement charge cannot be negative")
	}

	balanceBefore := decimal.Zero
	balanceAfter := decimal.Zero
	if relay.BalanceMode == RelayKeyBalanceModePrepaid && charge.GreaterThan(decimal.Zero) {
		balanceBefore, balanceAfter, err = debitRelayWallet(ctx, tx, dialectName, relay, charge)
		if err != nil {
			return err
		}
	} else if relay.Wallet != nil {
		balanceBefore = relay.Wallet.AvailableAmount
		balanceAfter = relay.Wallet.AvailableAmount
	}

	requestID := input.UsageLog.RequestID
	if input.Request != nil && input.Request.ID > 0 {
		requestID = input.Request.ID
	}

	if err := insertRelayLedgerEntry(ctx, tx, dialectName, relay, relayLedgerInsertInput{
		RequestID:      requestID,
		UsageLogID:     input.UsageLog.ID,
		Amount:         charge,
		BalanceBefore:  balanceBefore,
		BalanceAfter:   balanceAfter,
		UpstreamCost:   charge,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		if isUniqueConstraintError(err) {
			committed = true
			return tx.Commit()
		}

		return err
	}

	if err := upsertRelayDailyUsageSummary(ctx, tx, dialectName, relay, input.UsageLog, charge, requestID); err != nil {
		return err
	}
	if err := touchRelayKeyLastUsedAt(ctx, tx, dialectName, relay.RelayKeyID, time.Now().UTC()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit relay settlement transaction: %w", err)
	}
	committed = true

	return nil
}

type relayScanner interface {
	Scan(dest ...any) error
}

func scanRelayAccessContext(row relayScanner) (*RelayAccessContext, error) {
	var (
		relay                 RelayAccessContext
		keyStatus             string
		balanceMode           string
		providerType          string
		productStatus         string
		dailyRequestLimit     sql.NullInt64
		dailyTokenLimit       sql.NullInt64
		monthlyCostLimit      sql.NullString
		concurrencyLimit      sql.NullInt64
		expiresAt             sql.NullTime
		walletCurrency        sql.NullString
		walletAvailableAmount sql.NullString
		walletFrozenAmount    sql.NullString
		walletOverdraftLimit  sql.NullString
		walletVersion         sql.NullInt64
		dailyRequestCount     sql.NullInt64
		dailyTotalTokens      sql.NullInt64
		dailyTotalCharge      sql.NullString
	)

	if err := row.Scan(
		&relay.RelayKeyID,
		&relay.APIKeyID,
		&relay.ProjectID,
		&relay.ProductID,
		&keyStatus,
		&balanceMode,
		&dailyRequestLimit,
		&dailyTokenLimit,
		&monthlyCostLimit,
		&concurrencyLimit,
		&expiresAt,
		&relay.ProductCode,
		&relay.ProductName,
		&providerType,
		&productStatus,
		&walletCurrency,
		&walletAvailableAmount,
		&walletFrozenAmount,
		&walletOverdraftLimit,
		&walletVersion,
		&dailyRequestCount,
		&dailyTotalTokens,
		&dailyTotalCharge,
	); err != nil {
		return nil, err
	}

	relay.Status = RelayKeyStatus(keyStatus)
	relay.BalanceMode = RelayKeyBalanceMode(balanceMode)
	relay.ProviderType = RelayProductProviderType(providerType)
	relay.ProductStatus = RelayProductStatus(productStatus)
	if expiresAt.Valid {
		relay.ExpiresAt = &expiresAt.Time
	}
	if dailyRequestLimit.Valid {
		relay.Quota.DailyRequestLimit = &dailyRequestLimit.Int64
	}
	if dailyTokenLimit.Valid {
		relay.Quota.DailyTokenLimit = &dailyTokenLimit.Int64
	}
	if monthlyCostLimit.Valid && strings.TrimSpace(monthlyCostLimit.String) != "" {
		value, err := decimal.NewFromString(monthlyCostLimit.String)
		if err != nil {
			return nil, fmt.Errorf("invalid relay monthly cost limit %q: %w", monthlyCostLimit.String, err)
		}
		relay.Quota.MonthlyCostLimit = &value
	}
	if concurrencyLimit.Valid {
		relay.Quota.ConcurrencyLimit = &concurrencyLimit.Int64
	}
	if dailyRequestCount.Valid {
		relay.DailyUsage.RequestCount = dailyRequestCount.Int64
	}
	if dailyTotalTokens.Valid {
		relay.DailyUsage.TotalTokens = dailyTotalTokens.Int64
	}
	if dailyTotalCharge.Valid && strings.TrimSpace(dailyTotalCharge.String) != "" {
		value, err := decimal.NewFromString(dailyTotalCharge.String)
		if err != nil {
			return nil, fmt.Errorf("invalid relay daily total charge %q: %w", dailyTotalCharge.String, err)
		}
		relay.DailyUsage.TotalCharge = value
	}
	if walletCurrency.Valid {
		wallet := &RelayWalletSnapshot{
			Currency: walletCurrency.String,
			Version:  walletVersion.Int64,
		}
		var err error
		wallet.AvailableAmount, err = parseRelayDecimal(walletAvailableAmount, "wallet available amount")
		if err != nil {
			return nil, err
		}
		wallet.FrozenAmount, err = parseRelayDecimal(walletFrozenAmount, "wallet frozen amount")
		if err != nil {
			return nil, err
		}
		wallet.OverdraftLimit, err = parseRelayDecimal(walletOverdraftLimit, "wallet overdraft limit")
		if err != nil {
			return nil, err
		}
		relay.Wallet = wallet
	}

	return &relay, nil
}

func parseRelayDecimal(value sql.NullString, label string) (decimal.Decimal, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return decimal.Zero, nil
	}

	parsed, err := decimal.NewFromString(value.String)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid relay %s %q: %w", label, value.String, err)
	}

	return parsed, nil
}

func parseRelayStringList(value sql.NullString) ([]string, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}

	var out []string
	if err := json.Unmarshal([]byte(value.String), &out); err != nil {
		return nil, fmt.Errorf("invalid relay string list %q: %w", value.String, err)
	}

	return out, nil
}

func parseRelayModelFilter(value sql.NullString) []string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	var direct []string
	if err := json.Unmarshal([]byte(value.String), &direct); err == nil {
		return direct
	}

	var object map[string][]string
	if err := json.Unmarshal([]byte(value.String), &object); err != nil {
		return nil
	}
	for _, key := range []string{"models", "allowedModels", "patterns", "include"} {
		if models := object[key]; len(models) > 0 {
			return models
		}
	}

	return nil
}

func relaySQLDB(client *ent.Client) (*sql.DB, string, error) {
	if client == nil {
		return nil, "", fmt.Errorf("relay ent client is nil")
	}

	drv := client.Driver()
	sqlDriver, ok := drv.(*entsql.Driver)
	if !ok {
		return nil, "", fmt.Errorf("relay services require *sql.Driver, got %T", drv)
	}

	return sqlDriver.DB(), sqlDriver.Dialect(), nil
}

func relayPlaceholder(dialectName string, index int) string {
	if dialectName == dialect.Postgres {
		return fmt.Sprintf("$%d", index)
	}

	return "?"
}

func relayUTCDate(ts time.Time) time.Time {
	year, month, day := ts.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func relayLedgerEntryExists(ctx context.Context, tx *sql.Tx, dialectName, idempotencyKey string) (bool, error) {
	query := fmt.Sprintf("SELECT id FROM relay_wallet_ledger_entries WHERE idempotency_key = %s", relayPlaceholder(dialectName, 1))
	var id int
	err := tx.QueryRowContext(ctx, query, idempotencyKey).Scan(&id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check relay settlement idempotency: %w", err)
}

func debitRelayWallet(ctx context.Context, tx *sql.Tx, dialectName string, relay *RelayAccessContext, charge decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	lockClause := ""
	if dialectName == dialect.Postgres {
		lockClause = " FOR UPDATE"
	}
	query := fmt.Sprintf("SELECT available_amount, overdraft_limit, version FROM relay_wallets WHERE relay_key_id = %s%s", relayPlaceholder(dialectName, 1), lockClause)

	var (
		availableRaw sql.NullString
		overdraftRaw sql.NullString
		version      int64
	)
	if err := tx.QueryRowContext(ctx, query, relay.RelayKeyID).Scan(&availableRaw, &overdraftRaw, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, decimal.Zero, fmt.Errorf("%w: relay wallet is missing", ErrRelayInsufficientBalance)
		}

		return decimal.Zero, decimal.Zero, fmt.Errorf("failed to load relay wallet: %w", err)
	}

	available, err := parseRelayDecimal(availableRaw, "wallet available amount")
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	overdraft, err := parseRelayDecimal(overdraftRaw, "wallet overdraft limit")
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	if available.Add(overdraft).LessThan(charge) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("%w: relay wallet balance is lower than settlement charge", ErrRelayInsufficientBalance)
	}

	after := available.Sub(charge)
	update := fmt.Sprintf(
		"UPDATE relay_wallets SET available_amount = %s, version = version + 1, updated_at = %s WHERE relay_key_id = %s AND version = %s",
		relayPlaceholder(dialectName, 1),
		relayPlaceholder(dialectName, 2),
		relayPlaceholder(dialectName, 3),
		relayPlaceholder(dialectName, 4),
	)
	result, err := tx.ExecContext(ctx, update, after.String(), time.Now().UTC(), relay.RelayKeyID, version)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("failed to debit relay wallet: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("failed to inspect relay wallet debit result: %w", err)
	}
	if rows != 1 {
		return decimal.Zero, decimal.Zero, fmt.Errorf("relay wallet changed concurrently")
	}

	return available, after, nil
}

type relayLedgerInsertInput struct {
	RequestID      int
	UsageLogID     int
	Amount         decimal.Decimal
	BalanceBefore  decimal.Decimal
	BalanceAfter   decimal.Decimal
	UpstreamCost   decimal.Decimal
	IdempotencyKey string
}

func insertRelayLedgerEntry(ctx context.Context, tx *sql.Tx, dialectName string, relay *RelayAccessContext, input relayLedgerInsertInput) error {
	columns := []string{
		"relay_key_id",
		"project_id",
		"request_id",
		"usage_log_id",
		"direction",
		"scene",
		"amount",
		"balance_before",
		"balance_after",
		"upstream_cost",
		"price_snapshot",
		"idempotency_key",
		"remark",
	}
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = relayPlaceholder(dialectName, i+1)
	}

	query := fmt.Sprintf("INSERT INTO relay_wallet_ledger_entries (%s) VALUES (%s)", strings.Join(columns, ","), strings.Join(placeholders, ","))
	_, err := tx.ExecContext(ctx, query,
		relay.RelayKeyID,
		relay.ProjectID,
		nullableInt(input.RequestID),
		nullableInt(input.UsageLogID),
		"debit",
		"consume",
		input.Amount.String(),
		input.BalanceBefore.String(),
		input.BalanceAfter.String(),
		input.UpstreamCost.String(),
		[]byte("{}"),
		input.IdempotencyKey,
		"usage settlement",
	)
	if err != nil {
		return fmt.Errorf("failed to insert relay wallet ledger entry: %w", err)
	}

	return nil
}

func upsertRelayDailyUsageSummary(ctx context.Context, tx *sql.Tx, dialectName string, relay *RelayAccessContext, usageLog *ent.UsageLog, charge decimal.Decimal, requestID int) error {
	statDate := relayUTCDate(time.Now())
	if dialectName == dialect.MySQL {
		query := `INSERT INTO relay_daily_usage_summaries (relay_key_id, project_id, stat_date, request_count, total_tokens, total_charge, total_upstream_cost, last_request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
request_count = request_count + VALUES(request_count),
total_tokens = total_tokens + VALUES(total_tokens),
total_charge = CAST(total_charge AS DECIMAL(36,18)) + CAST(VALUES(total_charge) AS DECIMAL(36,18)),
total_upstream_cost = CAST(total_upstream_cost AS DECIMAL(36,18)) + CAST(VALUES(total_upstream_cost) AS DECIMAL(36,18)),
last_request_id = VALUES(last_request_id),
updated_at = CURRENT_TIMESTAMP`
		_, err := tx.ExecContext(ctx, query, relay.RelayKeyID, relay.ProjectID, statDate, 1, usageLog.TotalTokens, charge.String(), charge.String(), nullableInt(requestID))
		if err != nil {
			return fmt.Errorf("failed to upsert relay daily usage summary: %w", err)
		}
		return nil
	}

	totalChargeExpr := "CAST(relay_daily_usage_summaries.total_charge AS NUMERIC) + CAST(excluded.total_charge AS NUMERIC)"
	totalUpstreamCostExpr := "CAST(relay_daily_usage_summaries.total_upstream_cost AS NUMERIC) + CAST(excluded.total_upstream_cost AS NUMERIC)"
	if dialectName == dialect.Postgres {
		totalChargeExpr = "(CAST(relay_daily_usage_summaries.total_charge AS NUMERIC) + CAST(excluded.total_charge AS NUMERIC))::text"
		totalUpstreamCostExpr = "(CAST(relay_daily_usage_summaries.total_upstream_cost AS NUMERIC) + CAST(excluded.total_upstream_cost AS NUMERIC))::text"
	}

	query := fmt.Sprintf(`INSERT INTO relay_daily_usage_summaries (relay_key_id, project_id, stat_date, request_count, total_tokens, total_charge, total_upstream_cost, last_request_id)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
ON CONFLICT(relay_key_id, stat_date) DO UPDATE SET
request_count = relay_daily_usage_summaries.request_count + excluded.request_count,
total_tokens = relay_daily_usage_summaries.total_tokens + excluded.total_tokens,
total_charge = %s,
total_upstream_cost = %s,
last_request_id = excluded.last_request_id,
updated_at = CURRENT_TIMESTAMP`,
		relayPlaceholder(dialectName, 1),
		relayPlaceholder(dialectName, 2),
		relayPlaceholder(dialectName, 3),
		relayPlaceholder(dialectName, 4),
		relayPlaceholder(dialectName, 5),
		relayPlaceholder(dialectName, 6),
		relayPlaceholder(dialectName, 7),
		relayPlaceholder(dialectName, 8),
		totalChargeExpr,
		totalUpstreamCostExpr,
	)
	_, err := tx.ExecContext(ctx, query, relay.RelayKeyID, relay.ProjectID, statDate, 1, usageLog.TotalTokens, charge.String(), charge.String(), nullableInt(requestID))
	if err != nil {
		return fmt.Errorf("failed to upsert relay daily usage summary: %w", err)
	}

	return nil
}

func touchRelayKeyLastUsedAt(ctx context.Context, tx *sql.Tx, dialectName string, relayKeyID int, now time.Time) error {
	query := fmt.Sprintf("UPDATE relay_keys SET last_used_at = %s, updated_at = %s WHERE id = %s", relayPlaceholder(dialectName, 1), relayPlaceholder(dialectName, 2), relayPlaceholder(dialectName, 3))
	_, err := tx.ExecContext(ctx, query, now, now, relayKeyID)
	if err != nil {
		return fmt.Errorf("failed to touch relay key last used time: %w", err)
	}

	return nil
}

func nullableInt(value int) any {
	if value <= 0 {
		return nil
	}

	return value
}

func isMissingRelayTableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "no such table") || strings.Contains(message, "doesn't exist") || strings.Contains(message, "does not exist")
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicate key") || strings.Contains(message, "duplicate entry")
}
