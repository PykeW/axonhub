package biz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/enttest"
	"github.com/looplj/axonhub/internal/objects"
)

func TestRelayAccessService_LoadContextByAPIKeyOrdinaryBypass(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, "INSERT INTO api_keys (id, key, name, type, status, scopes, profiles, project_id, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)", 1001, "ordinary-key", "ordinary", "user", "enabled", `["read_channels"]`, `{}`, 7)

	svc := NewRelayAccessService(RelayAccessServiceParams{Ent: client, Router: NewRelayRouterService(client)})
	relay, err := svc.LoadContextByAPIKey(ctx, 1001)
	require.NoError(t, err)
	require.Nil(t, relay)
}

func TestRelayAccessService_CheckAccessValidation(t *testing.T) {
	svc := NewRelayAccessService(RelayAccessServiceParams{})
	now := time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC)
	dailyRequestLimit := int64(3)
	dailyTokenLimit := int64(100)
	zeroConcurrencyLimit := int64(0)
	positiveConcurrencyLimit := int64(1)

	require.NoError(t, svc.CheckAccess(context.Background(), nil, RelayAccessCheckInput{Now: now}))

	t.Run("project scope mismatch", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{APIKeyID: 10, ProjectID: 7, Status: RelayKeyStatusActive}, RelayAccessCheckInput{APIKey: &ent.APIKey{ID: 10, ProjectID: 8}, Now: now})
		require.ErrorIs(t, err, ErrRelayAccessDenied)
	})

	t.Run("disabled key", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{Status: RelayKeyStatusSuspended}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayAccessDenied)
	})

	t.Run("expired key", func(t *testing.T) {
		expiresAt := now.Add(-time.Minute)
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{Status: RelayKeyStatusActive, ExpiresAt: &expiresAt}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayAccessDenied)
	})

	t.Run("insufficient balance", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModePrepaid,
			Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.Zero, OverdraftLimit: decimal.Zero},
		}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayInsufficientBalance)
	})

	t.Run("daily request limit", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{DailyRequestLimit: &dailyRequestLimit},
			DailyUsage:  RelayUsageSnapshot{RequestCount: dailyRequestLimit},
		}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayQuotaExceeded)
	})

	t.Run("daily token limit", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{DailyTokenLimit: &dailyTokenLimit},
			DailyUsage:  RelayUsageSnapshot{TotalTokens: dailyTokenLimit},
		}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayQuotaExceeded)
	})

	t.Run("zero concurrency limit", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{ConcurrencyLimit: &zeroConcurrencyLimit},
		}, RelayAccessCheckInput{Now: now})
		require.ErrorIs(t, err, ErrRelayQuotaExceeded)
	})

	t.Run("positive concurrency limit allows access preflight", func(t *testing.T) {
		err := svc.CheckAccess(context.Background(), &RelayAccessContext{
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{ConcurrencyLimit: &positiveConcurrencyLimit},
		}, RelayAccessCheckInput{Now: now})
		require.NoError(t, err)
	})
}

func TestRelayRouterService_ListActiveChannelIDs(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	ch, err := client.Channel.Create().
		SetType(channel.TypeCodex).
		SetName("relay codex").
		SetStatus(channel.StatusEnabled).
		SetCredentials(objects.ChannelCredentials{}).
		SetSupportedModels([]string{"o3"}).
		SetDefaultTestModel("o3").
		Save(ctx)
	require.NoError(t, err)

	relayServicesTestExec(t, db, `INSERT INTO relay_products (id, code, name, provider_type, access_mode, billing_mode, status, currency, allowed_models, request_timeout_seconds, list_price_config, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`, 2001, "codex-shared", "Codex Shared", "codex", "shared_capacity", "prepaid", "active", "USD", `["o3","gpt-*"]`, 120, `{}`)
	relayServicesTestExec(t, db, `INSERT INTO relay_product_channels (id, product_id, channel_id, priority, weight, status, allow_fallback, model_filter, max_inflight)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, 3001, 2001, ch.ID, 10, 100, "active", true, `["o3"]`, 5)

	svc := NewRelayRouterService(client)
	ids, err := svc.ListActiveChannelIDs(ctx, 2001, "o3")
	require.NoError(t, err)
	require.Equal(t, []int{ch.ID}, ids)

	ids, err = svc.ListActiveChannelIDs(ctx, 2001, "claude-3-5-sonnet")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestRelaySettlementService_SettleUsageIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`, 4001, 5001, 7, 2001, "sub key", "active", "prepaid")
	relayServicesTestExec(t, db, `INSERT INTO relay_wallets (id, relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, 5001, 4001, 7, "USD", "10", "0", "0", 1)

	charge := 1.25
	relay := &RelayAccessContext{
		RelayKeyID:  4001,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModePrepaid,
		Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.NewFromInt(10)},
	}
	usageLog := &ent.UsageLog{ID: 6001, RequestID: 7001, TotalTokens: 123, TotalCost: &charge}

	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}))
	require.NoError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}))

	assertRelaySettlementChargedOnce(t, db, relay.RelayKeyID, "8.75", 123, "1.25")
}

func TestRelaySettlementService_SettleUsageConcurrentIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`, 4002, 5002, 7, 2001, "concurrent sub key", "active", "prepaid")
	relayServicesTestExec(t, db, `INSERT INTO relay_wallets (id, relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, 5002, 4002, 7, "USD", "10", "0", "0", 1)

	charge := 1.25
	relay := &RelayAccessContext{
		RelayKeyID:  4002,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModePrepaid,
		Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.NewFromInt(10)},
	}
	usageLog := &ent.UsageLog{ID: 6002, RequestID: 7002, TotalTokens: 123, TotalCost: &charge}
	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})

	const workers = 8
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assertRelaySettlementChargedOnce(t, db, relay.RelayKeyID, "8.75", 123, "1.25")
}

func TestRelaySettlementService_SettleUsageMonthlyHardCapExactAllowed(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, monthly_cost_limit, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`, 4003, 5003, 7, 2001, "quota exact cap", "active", "quota_only", "1")
	relayServicesTestExec(t, db, `INSERT INTO relay_daily_usage_summaries (relay_key_id, project_id, stat_date, request_count, total_tokens, total_charge, total_upstream_cost)
VALUES (?, ?, ?, ?, ?, ?, ?)`, 4003, 7, relayUTCDate(time.Now()), 1, 50, "0.75", "0.75")

	charge := 0.25
	relay := &RelayAccessContext{
		RelayKeyID:  4003,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModeQuotaOnly,
	}
	usageLog := &ent.UsageLog{ID: 6003, RequestID: 7003, TotalTokens: 25, TotalCost: &charge}

	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}))

	assertRelayLedgerCount(t, db, relay.RelayKeyID, 1)
	assertRelayUsageSummary(t, db, relay.RelayKeyID, 2, 75, "1")
}

func TestRelaySettlementService_SettleUsageMonthlyHardCapRejectsOverflow(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, monthly_cost_limit, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`, 4004, 5004, 7, 2001, "cap overflow", "active", "prepaid", "1")
	relayServicesTestExec(t, db, `INSERT INTO relay_wallets (id, relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, 5004, 4004, 7, "USD", "10", "0", "0", 1)

	charge := 1.01
	relayLimitFallback := decimal.NewFromInt(100)
	relay := &RelayAccessContext{
		RelayKeyID:  4004,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModePrepaid,
		Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.NewFromInt(10)},
		Quota:       RelayKeyQuotaSnapshot{MonthlyCostLimit: &relayLimitFallback},
	}
	usageLog := &ent.UsageLog{ID: 6004, RequestID: 7004, TotalTokens: 101, TotalCost: &charge}

	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})
	requireRelayError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}), ErrRelayQuotaExceeded)

	assertRelayWalletBalance(t, db, relay.RelayKeyID, "10")
	assertRelayLedgerCount(t, db, relay.RelayKeyID, 0)
	assertRelayNoUsageSummary(t, db, relay.RelayKeyID)
}

func TestRelaySettlementService_SettleUsageIdempotentRetryAfterMonthlyHardCap(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, monthly_cost_limit, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`, 4005, 5005, 7, 2001, "retry at cap", "active", "prepaid", "1.25")
	relayServicesTestExec(t, db, `INSERT INTO relay_wallets (id, relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, 5005, 4005, 7, "USD", "10", "0", "0", 1)

	charge := 1.25
	relay := &RelayAccessContext{
		RelayKeyID:  4005,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModePrepaid,
		Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.NewFromInt(10)},
	}
	usageLog := &ent.UsageLog{ID: 6005, RequestID: 7005, TotalTokens: 125, TotalCost: &charge}

	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}))
	require.NoError(t, svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog}))

	assertRelaySettlementChargedOnce(t, db, relay.RelayKeyID, "8.75", 125, "1.25")
}

func TestRelaySettlementService_SettleUsageDifferentConcurrentMonthlyHardCap(t *testing.T) {
	ctx := context.Background()
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	relayServicesTestExec(t, db, `INSERT INTO relay_keys (id, api_key_id, project_id, product_id, display_name, status, balance_mode, monthly_cost_limit, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`, 4006, 5006, 7, 2001, "different usage cap", "active", "prepaid", "1.25")
	relayServicesTestExec(t, db, `INSERT INTO relay_wallets (id, relay_key_id, project_id, currency, available_amount, frozen_amount, overdraft_limit, version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, 5006, 4006, 7, "USD", "10", "0", "0", 1)

	charge := 0.75
	relay := &RelayAccessContext{
		RelayKeyID:  4006,
		ProjectID:   7,
		BalanceMode: RelayKeyBalanceModePrepaid,
		Wallet:      &RelayWalletSnapshot{AvailableAmount: decimal.NewFromInt(10)},
	}
	usageLogs := []*ent.UsageLog{
		{ID: 6006, RequestID: 7006, TotalTokens: 75, TotalCost: &charge},
		{ID: 6007, RequestID: 7007, TotalTokens: 75, TotalCost: &charge},
	}
	svc := NewRelaySettlementService(RelaySettlementServiceParams{Ent: client})

	start := make(chan struct{})
	errs := make(chan error, len(usageLogs))
	var wg sync.WaitGroup
	for _, usageLog := range usageLogs {
		usageLog := usageLog
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- svc.SettleUsage(ctx, relay, RelayUsageSettlementInput{UsageLog: usageLog})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	quotaExceeded := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, ErrRelayQuotaExceeded) {
			quotaExceeded++
			continue
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, quotaExceeded)
	assertRelaySettlementChargedOnce(t, db, relay.RelayKeyID, "9.25", 75, "0.75")
}

func assertRelaySettlementChargedOnce(t *testing.T, db *sql.DB, relayKeyID int, expectedBalance string, expectedTokens int64, expectedCharge string) {
	t.Helper()
	assertRelayWalletBalance(t, db, relayKeyID, expectedBalance)
	assertRelayLedgerCount(t, db, relayKeyID, 1)
	assertRelayUsageSummary(t, db, relayKeyID, 1, expectedTokens, expectedCharge)
}

func assertRelayWalletBalance(t *testing.T, db *sql.DB, relayKeyID int, expectedBalance string) {
	t.Helper()
	var walletBalance string
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT available_amount FROM relay_wallets WHERE relay_key_id = ?", relayKeyID).Scan(&walletBalance))
	require.Equal(t, expectedBalance, walletBalance)
}

func assertRelayLedgerCount(t *testing.T, db *sql.DB, relayKeyID int, expectedCount int) {
	t.Helper()
	var ledgerCount int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM relay_wallet_ledger_entries WHERE relay_key_id = ?", relayKeyID).Scan(&ledgerCount))
	require.Equal(t, expectedCount, ledgerCount)
}

func assertRelayUsageSummary(t *testing.T, db *sql.DB, relayKeyID int, expectedRequestCount, expectedTokens int64, expectedCharge string) {
	t.Helper()
	var requestCount, totalTokens int64
	var totalChargeRaw string
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT request_count, total_tokens, total_charge FROM relay_daily_usage_summaries WHERE relay_key_id = ?", relayKeyID).Scan(&requestCount, &totalTokens, &totalChargeRaw))
	require.Equal(t, expectedRequestCount, requestCount)
	require.Equal(t, expectedTokens, totalTokens)
	totalCharge, err := decimal.NewFromString(totalChargeRaw)
	require.NoError(t, err)
	expectedTotalCharge, err := decimal.NewFromString(expectedCharge)
	require.NoError(t, err)
	require.True(t, totalCharge.Equal(expectedTotalCharge), "expected total_charge %s, got %s", expectedTotalCharge, totalCharge)
}

func assertRelayNoUsageSummary(t *testing.T, db *sql.DB, relayKeyID int) {
	t.Helper()
	var summaryCount int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM relay_daily_usage_summaries WHERE relay_key_id = ?", relayKeyID).Scan(&summaryCount))
	require.Equal(t, 0, summaryCount)
}

func newRelayServicesTestClient(t *testing.T) *ent.Client {
	t.Helper()
	client := enttest.NewEntClient(t, dialect.SQLite, fmt.Sprintf("file:relay_services_%d?mode=memory&cache=shared&_fk=0", time.Now().UnixNano()))
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	installRelayServicesTestSchema(t, client)
	return client
}

func relayServicesTestDB(t *testing.T, client *ent.Client) *sql.DB {
	t.Helper()
	db, _, err := relaySQLDB(client)
	require.NoError(t, err)
	return db
}

func installRelayServicesTestSchema(t *testing.T, client *ent.Client) {
	t.Helper()
	db := relayServicesTestDB(t, client)
	for _, stmt := range relayServicesTestSchemaStatements() {
		relayServicesTestExec(t, db, stmt)
	}
}

func relayServicesTestExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	require.NoError(t, err)
}

func relayServicesTestSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS relay_products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at INTEGER DEFAULT 0,
			code TEXT NOT NULL,
			name TEXT NOT NULL,
			provider_type TEXT NOT NULL,
			access_mode TEXT NOT NULL DEFAULT 'shared_capacity',
			billing_mode TEXT NOT NULL DEFAULT 'prepaid',
			status TEXT NOT NULL DEFAULT 'draft',
			currency TEXT NOT NULL DEFAULT 'USD',
			allowed_models TEXT,
			request_timeout_seconds INTEGER DEFAULT 0,
			list_price_config TEXT,
			UNIQUE(code, deleted_at)
		)`,
		`CREATE TABLE IF NOT EXISTS relay_product_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			product_id INTEGER NOT NULL,
			channel_id INTEGER NOT NULL,
			priority INTEGER NOT NULL DEFAULT 100,
			weight INTEGER NOT NULL DEFAULT 100,
			status TEXT NOT NULL DEFAULT 'active',
			allow_fallback BOOLEAN NOT NULL DEFAULT TRUE,
			model_filter TEXT,
			max_inflight INTEGER,
			UNIQUE(product_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS relay_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at INTEGER DEFAULT 0,
			api_key_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			owner_user_id INTEGER,
			display_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			balance_mode TEXT NOT NULL DEFAULT 'prepaid',
			daily_request_limit INTEGER,
			daily_token_limit INTEGER,
			monthly_cost_limit TEXT,
			concurrency_limit INTEGER,
			expires_at DATETIME,
			last_used_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS relay_wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			relay_key_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			currency TEXT NOT NULL DEFAULT 'USD',
			available_amount TEXT NOT NULL DEFAULT '0',
			frozen_amount TEXT NOT NULL DEFAULT '0',
			overdraft_limit TEXT NOT NULL DEFAULT '0',
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS relay_wallet_ledger_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			relay_key_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			request_id INTEGER,
			usage_log_id INTEGER,
			direction TEXT NOT NULL,
			scene TEXT NOT NULL,
			amount TEXT NOT NULL,
			balance_before TEXT NOT NULL,
			balance_after TEXT NOT NULL,
			upstream_cost TEXT,
			price_snapshot TEXT,
			idempotency_key TEXT NOT NULL,
			operator_user_id INTEGER,
			remark TEXT,
			UNIQUE(idempotency_key)
		)`,
		`CREATE TABLE IF NOT EXISTS relay_daily_usage_summaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			relay_key_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			stat_date DATETIME NOT NULL,
			request_count INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			total_charge TEXT NOT NULL DEFAULT '0',
			total_upstream_cost TEXT NOT NULL DEFAULT '0',
			last_request_id INTEGER,
			UNIQUE(relay_key_id, stat_date)
		)`,
	}
}

func requireRelayError(t *testing.T, err error, target error) {
	t.Helper()
	require.True(t, errors.Is(err, target), "expected %v, got %v", target, err)
}
