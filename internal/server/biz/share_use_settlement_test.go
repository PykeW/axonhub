package biz

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	entapikey "github.com/looplj/axonhub/internal/ent/apikey"
	entchannel "github.com/looplj/axonhub/internal/ent/channel"
	entproject "github.com/looplj/axonhub/internal/ent/project"
	entrequest "github.com/looplj/axonhub/internal/ent/request"
	entusagelog "github.com/looplj/axonhub/internal/ent/usagelog"
	"github.com/looplj/axonhub/internal/objects"
)

func TestShareUseSettlementService_SettleUsageCrossUserSuccess(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)
	installShareUseSettlementTestSchema(t, db)
	ctx = ent.NewContext(ctx, client)

	caller := createShareUseSettlementUser(t, client, ctx, "caller-success@example.com")
	owner := createShareUseSettlementUser(t, client, ctx, "owner-success@example.com")
	project := createShareUseSettlementProject(t, client, ctx, "share-use-success-project")
	apiKey := createShareUseSettlementAPIKey(t, client, ctx, caller.ID, project.ID, "caller-success-key")
	channel := createShareUseSettlementChannel(t, client, ctx, owner.ID, "shared-success-channel")
	req := createShareUseSettlementRequest(t, client, ctx, project.ID, apiKey.ID, entrequest.StatusProcessing, "gpt-4")
	usageLog := createShareUseSettlementUsageLog(t, client, ctx, req.ID, project.ID, channel.ID, apiKey.ID, 1.25)

	settleCtx := contexts.WithAPIKey(ctx, apiKey)
	svc := NewShareUseSettlementService(ShareUseSettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(settleCtx, ShareUseUsageSettlementInput{UsageLog: usageLog, Request: req}))

	assertUserPointAccountSnapshot(t, db, caller.ID, "-1.25", "0", "1.25")
	assertUserPointAccountSnapshot(t, db, owner.ID, "1.25", "1.25", "0")
	assertUserPointLedgerCount(t, db, 2)
	assertUserPointLedgerEntry(t, db, caller.ID, "debit", "consume", usageLog.ID, "1.25")
	assertUserPointLedgerEntry(t, db, owner.ID, "credit", "contribution_reward", usageLog.ID, "1.25")
}

func TestShareUseSettlementService_SettleUsageOwnChannelSkips(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)
	installShareUseSettlementTestSchema(t, db)
	ctx = ent.NewContext(ctx, client)

	caller := createShareUseSettlementUser(t, client, ctx, "caller-own@example.com")
	project := createShareUseSettlementProject(t, client, ctx, "share-use-own-project")
	apiKey := createShareUseSettlementAPIKey(t, client, ctx, caller.ID, project.ID, "caller-own-key")
	channel := createShareUseSettlementChannel(t, client, ctx, caller.ID, "shared-own-channel")
	req := createShareUseSettlementRequest(t, client, ctx, project.ID, apiKey.ID, entrequest.StatusProcessing, "gpt-4")
	usageLog := createShareUseSettlementUsageLog(t, client, ctx, req.ID, project.ID, channel.ID, apiKey.ID, 1.25)

	settleCtx := contexts.WithAPIKey(ctx, apiKey)
	svc := NewShareUseSettlementService(ShareUseSettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(settleCtx, ShareUseUsageSettlementInput{UsageLog: usageLog, Request: req}))

	assertUserPointAccountCount(t, db, 0)
	assertUserPointLedgerCount(t, db, 0)
}

func TestShareUseSettlementService_SettleUsageFailedRequestSkips(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)
	installShareUseSettlementTestSchema(t, db)
	ctx = ent.NewContext(ctx, client)

	caller := createShareUseSettlementUser(t, client, ctx, "caller-failed@example.com")
	owner := createShareUseSettlementUser(t, client, ctx, "owner-failed@example.com")
	project := createShareUseSettlementProject(t, client, ctx, "share-use-failed-project")
	apiKey := createShareUseSettlementAPIKey(t, client, ctx, caller.ID, project.ID, "caller-failed-key")
	channel := createShareUseSettlementChannel(t, client, ctx, owner.ID, "shared-failed-channel")
	req := createShareUseSettlementRequest(t, client, ctx, project.ID, apiKey.ID, entrequest.StatusFailed, "gpt-4")
	usageLog := createShareUseSettlementUsageLog(t, client, ctx, req.ID, project.ID, channel.ID, apiKey.ID, 1.25)

	settleCtx := contexts.WithAPIKey(ctx, apiKey)
	svc := NewShareUseSettlementService(ShareUseSettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(settleCtx, ShareUseUsageSettlementInput{UsageLog: usageLog, Request: req}))

	assertUserPointAccountCount(t, db, 0)
	assertUserPointLedgerCount(t, db, 0)
}

func TestShareUseSettlementService_SettleUsageIdempotentRetry(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)
	installShareUseSettlementTestSchema(t, db)
	ctx = ent.NewContext(ctx, client)

	caller := createShareUseSettlementUser(t, client, ctx, "caller-idempotent@example.com")
	owner := createShareUseSettlementUser(t, client, ctx, "owner-idempotent@example.com")
	project := createShareUseSettlementProject(t, client, ctx, "share-use-idempotent-project")
	apiKey := createShareUseSettlementAPIKey(t, client, ctx, caller.ID, project.ID, "caller-idempotent-key")
	channel := createShareUseSettlementChannel(t, client, ctx, owner.ID, "shared-idempotent-channel")
	req := createShareUseSettlementRequest(t, client, ctx, project.ID, apiKey.ID, entrequest.StatusProcessing, "gpt-4")
	usageLog := createShareUseSettlementUsageLog(t, client, ctx, req.ID, project.ID, channel.ID, apiKey.ID, 1.25)

	settleCtx := contexts.WithAPIKey(ctx, apiKey)
	svc := NewShareUseSettlementService(ShareUseSettlementServiceParams{Ent: client})
	require.NoError(t, svc.SettleUsage(settleCtx, ShareUseUsageSettlementInput{UsageLog: usageLog, Request: req}))
	require.NoError(t, svc.SettleUsage(settleCtx, ShareUseUsageSettlementInput{UsageLog: usageLog, Request: req}))

	assertUserPointAccountSnapshot(t, db, caller.ID, "-1.25", "0", "1.25")
	assertUserPointAccountSnapshot(t, db, owner.ID, "1.25", "1.25", "0")
	assertUserPointLedgerCount(t, db, 2)
}

func createShareUseSettlementUser(t *testing.T, client *ent.Client, ctx context.Context, email string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
		SetEmail(email).
		SetPassword("test-password").
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createShareUseSettlementProject(t *testing.T, client *ent.Client, ctx context.Context, name string) *ent.Project {
	t.Helper()
	project, err := client.Project.Create().
		SetName(name).
		SetStatus(entproject.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return project
}

func createShareUseSettlementAPIKey(t *testing.T, client *ent.Client, ctx context.Context, userID, projectID int, key string) *ent.APIKey {
	t.Helper()
	apiKey, err := client.APIKey.Create().
		SetName(key).
		SetKey(key).
		SetUserID(userID).
		SetProjectID(projectID).
		SetType(entapikey.TypeUser).
		SetStatus(entapikey.StatusEnabled).
		SetScopes([]string{"read_channels", "write_requests"}).
		Save(ctx)
	require.NoError(t, err)
	return apiKey
}

func createShareUseSettlementChannel(t *testing.T, client *ent.Client, ctx context.Context, ownerUserID int, name string) *ent.Channel {
	t.Helper()
	channel, err := client.Channel.Create().
		SetType(entchannel.TypeOpenai).
		SetName(name).
		SetCredentials(objects.ChannelCredentials{APIKey: "owner-token"}).
		SetSupportedModels([]string{"gpt-4"}).
		SetDefaultTestModel("gpt-4").
		SetSettings(&objects.ChannelSettings{
			Share: &objects.ChannelShareSettings{
				OwnerUserID: &objects.GUID{Type: ent.TypeUser, ID: ownerUserID},
				Visibility:  objects.ChannelVisibilityShared,
			},
		}).
		Save(ctx)
	require.NoError(t, err)
	return channel
}

func createShareUseSettlementRequest(t *testing.T, client *ent.Client, ctx context.Context, projectID, apiKeyID int, status entrequest.Status, modelID string) *ent.Request {
	t.Helper()
	req, err := client.Request.Create().
		SetProjectID(projectID).
		SetAPIKeyID(apiKeyID).
		SetModelID(modelID).
		SetStatus(status).
		SetRequestBody(objects.JSONRawMessage([]byte(`{}`))).
		Save(ctx)
	require.NoError(t, err)
	return req
}

func createShareUseSettlementUsageLog(t *testing.T, client *ent.Client, ctx context.Context, requestID, projectID, channelID, apiKeyID int, totalCost float64) *ent.UsageLog {
	t.Helper()
	usageLog, err := client.UsageLog.Create().
		SetRequestID(requestID).
		SetAPIKeyID(apiKeyID).
		SetProjectID(projectID).
		SetChannelID(channelID).
		SetModelID("gpt-4").
		SetPromptTokens(10).
		SetCompletionTokens(5).
		SetTotalTokens(15).
		SetSource(entusagelog.SourceAPI).
		SetFormat("openai/chat_completions").
		SetTotalCost(totalCost).
		Save(ctx)
	require.NoError(t, err)
	return usageLog
}

func installShareUseSettlementTestSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range shareUseSettlementTestSchemaStatements() {
		relayServicesTestExec(t, db, stmt)
	}
}

func shareUseSettlementTestSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS user_point_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER NOT NULL,
			available_points TEXT NOT NULL DEFAULT '0',
			pending_points TEXT NOT NULL DEFAULT '0',
			frozen_points TEXT NOT NULL DEFAULT '0',
			lifetime_earned TEXT NOT NULL DEFAULT '0',
			lifetime_spent TEXT NOT NULL DEFAULT '0',
			version INTEGER NOT NULL DEFAULT 1,
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_point_ledger_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER NOT NULL,
			direction TEXT NOT NULL,
			scene TEXT NOT NULL,
			points TEXT NOT NULL,
			balance_before TEXT NOT NULL,
			balance_after TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			related_channel_id INTEGER,
			related_request_id INTEGER,
			related_usage_log_id INTEGER,
			related_api_key_id INTEGER,
			related_project_id INTEGER,
			conversion_rate_snapshot TEXT,
			settlement_status TEXT NOT NULL DEFAULT 'posted',
			remark TEXT,
			UNIQUE(idempotency_key)
		)`,
	}
}

func assertUserPointAccountSnapshot(t *testing.T, db *sql.DB, userID int, expectedAvailable, expectedEarned, expectedSpent string) {
	t.Helper()
	var available, earned, spent string
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT available_points, lifetime_earned, lifetime_spent FROM user_point_accounts WHERE user_id = ?", userID).Scan(&available, &earned, &spent))
	require.Equal(t, expectedAvailable, available)
	require.Equal(t, expectedEarned, earned)
	require.Equal(t, expectedSpent, spent)
}

func assertUserPointAccountCount(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM user_point_accounts").Scan(&count))
	require.Equal(t, expected, count)
}

func assertUserPointLedgerCount(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM user_point_ledger_entries").Scan(&count))
	require.Equal(t, expected, count)
}

func assertUserPointLedgerEntry(t *testing.T, db *sql.DB, userID int, direction, scene string, usageLogID int, expectedPoints string) {
	t.Helper()
	var points string
	query := "SELECT points FROM user_point_ledger_entries WHERE user_id = ? AND direction = ? AND scene = ? AND related_usage_log_id = ?"
	require.NoError(t, db.QueryRowContext(context.Background(), query, userID, direction, scene, usageLogID).Scan(&points), fmt.Sprintf("missing ledger row for user %d direction=%s scene=%s", userID, direction, scene))
	require.Equal(t, expectedPoints, points)
}
