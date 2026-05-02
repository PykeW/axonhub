package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"entgo.io/ent/dialect"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/request"
	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/ent/usagelog"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
)

type ShareUseSettlementServiceParams struct {
	fx.In

	Ent *ent.Client
}

type ShareUseUsageSettlementInput struct {
	UsageLog *ent.UsageLog
	Request  *ent.Request
}

type ShareUseSettlementRecorder interface {
	RecordShareUseUsage(ctx context.Context, input ShareUseUsageSettlementInput) error
}

// ShareUseSettlementService handles cross-user Share/Use point settlement.
type ShareUseSettlementService struct {
	*AbstractService
}

func NewShareUseSettlementService(params ShareUseSettlementServiceParams) *ShareUseSettlementService {
	return &ShareUseSettlementService{AbstractService: &AbstractService{db: params.Ent}}
}

func (s *ShareUseSettlementService) RecordShareUseUsage(ctx context.Context, input ShareUseUsageSettlementInput) error {
	return s.SettleUsage(ctx, input)
}

type shareUseSettlementContext struct {
	callerUserID           int
	ownerUserID            int
	apiKeyID               int
	projectID              int
	requestID              int
	usageLogID             int
	channelID              int
	points                 decimal.Decimal
	conversionRateSnapshot string
}

type userPointAccountSnapshot struct {
	UserID          int
	AvailablePoints decimal.Decimal
	PendingPoints   decimal.Decimal
	FrozenPoints    decimal.Decimal
	LifetimeEarned  decimal.Decimal
	LifetimeSpent   decimal.Decimal
	Version         int64
}

type userPointLedgerInsertInput struct {
	UserID                 int
	Direction              string
	Scene                  string
	Points                 decimal.Decimal
	BalanceBefore          decimal.Decimal
	BalanceAfter           decimal.Decimal
	IdempotencyKey         string
	RelatedChannelID       int
	RelatedRequestID       int
	RelatedUsageLogID      int
	RelatedAPIKeyID        int
	RelatedProjectID       int
	ConversionRateSnapshot string
	SettlementStatus       string
	Remark                 string
}

func (s *ShareUseSettlementService) SettleUsage(ctx context.Context, input ShareUseUsageSettlementInput) error {
	if s == nil || input.UsageLog == nil {
		return nil
	}
	if input.UsageLog.ID <= 0 {
		return fmt.Errorf("usage log id must be greater than 0 for share/use settlement")
	}
	if _, ok := contexts.GetRelayAuthContext(ctx); ok {
		// Relay/Sub-Key traffic is still settled by the legacy relay path.
		return nil
	}

	settlement, err := s.resolveSettlementContext(ctx, input)
	if err != nil || settlement == nil {
		return err
	}

	db, dialectName, err := relaySQLDB(s.db)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin share/use settlement transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	debitKey := shareUseDebitIdempotencyKey(settlement.usageLogID, settlement.callerUserID)
	creditKey := shareUseCreditIdempotencyKey(settlement.usageLogID, settlement.ownerUserID)

	exists, err := userPointLedgerEntryExists(ctx, tx, dialectName, debitKey)
	if err != nil {
		return err
	}
	if exists {
		committed = true
		return tx.Commit()
	}

	lockedUserIDs := shareUseLockedUserIDs(settlement.callerUserID, settlement.ownerUserID)
	accounts := make(map[int]*userPointAccountSnapshot, len(lockedUserIDs))
	for _, userID := range lockedUserIDs {
		account, err := loadUserPointAccountForUpdate(ctx, tx, dialectName, userID)
		if err != nil {
			return err
		}
		accounts[userID] = account
	}

	callerAccount := accounts[settlement.callerUserID]
	ownerAccount := accounts[settlement.ownerUserID]
	if callerAccount == nil || ownerAccount == nil {
		return fmt.Errorf("failed to lock caller/owner point accounts for share/use settlement")
	}

	callerBalanceBefore := callerAccount.AvailablePoints
	ownerBalanceBefore := ownerAccount.AvailablePoints

	callerAccount.AvailablePoints = callerAccount.AvailablePoints.Sub(settlement.points)
	callerAccount.LifetimeSpent = callerAccount.LifetimeSpent.Add(settlement.points)

	ownerAccount.AvailablePoints = ownerAccount.AvailablePoints.Add(settlement.points)
	ownerAccount.LifetimeEarned = ownerAccount.LifetimeEarned.Add(settlement.points)

	for _, userID := range lockedUserIDs {
		if err := updateUserPointAccount(ctx, tx, dialectName, accounts[userID]); err != nil {
			return err
		}
	}

	if err := insertUserPointLedgerEntry(ctx, tx, dialectName, userPointLedgerInsertInput{
		UserID:                 settlement.callerUserID,
		Direction:              "debit",
		Scene:                  "consume",
		Points:                 settlement.points,
		BalanceBefore:          callerBalanceBefore,
		BalanceAfter:           callerAccount.AvailablePoints,
		IdempotencyKey:         debitKey,
		RelatedChannelID:       settlement.channelID,
		RelatedRequestID:       settlement.requestID,
		RelatedUsageLogID:      settlement.usageLogID,
		RelatedAPIKeyID:        settlement.apiKeyID,
		RelatedProjectID:       settlement.projectID,
		ConversionRateSnapshot: settlement.conversionRateSnapshot,
		SettlementStatus:       "posted",
		Remark:                 "share/use shared channel consume",
	}); err != nil {
		if isUniqueConstraintError(err) {
			return nil
		}

		return err
	}

	if err := insertUserPointLedgerEntry(ctx, tx, dialectName, userPointLedgerInsertInput{
		UserID:                 settlement.ownerUserID,
		Direction:              "credit",
		Scene:                  "contribution_reward",
		Points:                 settlement.points,
		BalanceBefore:          ownerBalanceBefore,
		BalanceAfter:           ownerAccount.AvailablePoints,
		IdempotencyKey:         creditKey,
		RelatedChannelID:       settlement.channelID,
		RelatedRequestID:       settlement.requestID,
		RelatedUsageLogID:      settlement.usageLogID,
		RelatedAPIKeyID:        settlement.apiKeyID,
		RelatedProjectID:       settlement.projectID,
		ConversionRateSnapshot: settlement.conversionRateSnapshot,
		SettlementStatus:       "posted",
		Remark:                 "share/use shared channel contribution reward",
	}); err != nil {
		if isUniqueConstraintError(err) {
			return nil
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit share/use settlement transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *ShareUseSettlementService) resolveSettlementContext(ctx context.Context, input ShareUseUsageSettlementInput) (*shareUseSettlementContext, error) {
	if input.UsageLog == nil || input.UsageLog.Source != usagelog.SourceAPI {
		return nil, nil
	}
	if input.Request != nil {
		switch input.Request.Status {
		case request.StatusFailed, request.StatusCanceled:
			return nil, nil
		}
	}

	callerUserID, apiKeyID, projectID, ok := resolveShareUseCaller(ctx, input)
	if !ok {
		return nil, nil
	}

	ownerUserID, shared, err := s.loadSharedChannelOwner(ctx, input.UsageLog.ChannelID)
	if err != nil {
		return nil, err
	}
	if !shared || ownerUserID <= 0 || ownerUserID == callerUserID {
		return nil, nil
	}

	points, snapshot, ok, err := deriveShareUseSettlementPoints(input.UsageLog)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	requestID := input.UsageLog.RequestID
	if input.Request != nil && input.Request.ID > 0 {
		requestID = input.Request.ID
	}

	if projectID <= 0 {
		if input.Request != nil && input.Request.ProjectID > 0 {
			projectID = input.Request.ProjectID
		} else if input.UsageLog.ProjectID > 0 {
			projectID = input.UsageLog.ProjectID
		}
	}

	return &shareUseSettlementContext{
		callerUserID:           callerUserID,
		ownerUserID:            ownerUserID,
		apiKeyID:               apiKeyID,
		projectID:              projectID,
		requestID:              requestID,
		usageLogID:             input.UsageLog.ID,
		channelID:              input.UsageLog.ChannelID,
		points:                 points,
		conversionRateSnapshot: snapshot,
	}, nil
}

func resolveShareUseCaller(ctx context.Context, input ShareUseUsageSettlementInput) (userID, apiKeyID, projectID int, ok bool) {
	if apiKey, exists := contexts.GetAPIKey(ctx); exists && apiKey != nil {
		if apiKey.UserID > 0 {
			return apiKey.UserID, apiKey.ID, apiKey.ProjectID, true
		}
		apiKeyID = apiKey.ID
		projectID = apiKey.ProjectID
	}

	if user, exists := contexts.GetUser(ctx); exists && user != nil && user.ID > 0 {
		return user.ID, apiKeyID, projectID, true
	}

	if input.Request != nil && input.Request.APIKeyID > 0 {
		apiKeyID = input.Request.APIKeyID
	}
	if projectID <= 0 && input.Request != nil {
		projectID = input.Request.ProjectID
	}
	if projectID <= 0 && input.UsageLog != nil {
		projectID = input.UsageLog.ProjectID
	}

	return 0, apiKeyID, projectID, false
}

func deriveShareUseSettlementPoints(usageLog *ent.UsageLog) (decimal.Decimal, string, bool, error) {
	if usageLog == nil || usageLog.TotalCost == nil {
		return decimal.Zero, "", false, nil
	}

	points := decimal.NewFromFloat(*usageLog.TotalCost)
	if points.IsNegative() {
		return decimal.Zero, "", false, fmt.Errorf("share/use settlement points cannot be negative")
	}
	if !points.GreaterThan(decimal.Zero) {
		return decimal.Zero, "", false, nil
	}

	snapshot := map[string]any{
		"mode":   "1_to_1_total_cost_to_points",
		"source": "usage_log.total_cost",
	}
	if usageLog.CostPriceReferenceID != "" {
		snapshot["costPriceReferenceID"] = usageLog.CostPriceReferenceID
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return decimal.Zero, "", false, fmt.Errorf("failed to marshal share/use conversion snapshot: %w", err)
	}

	return points, string(payload), true, nil
}

func (s *ShareUseSettlementService) loadSharedChannelOwner(ctx context.Context, channelID int) (int, bool, error) {
	if channelID <= 0 {
		return 0, false, nil
	}

	loaded, err := authz.RunWithSystemBypass(ctx, "share-use-settlement-load-channel", func(bypassCtx context.Context) (*ent.Channel, error) {
		bypassCtx = schematype.SkipSoftDelete(bypassCtx)
		return s.entFromContext(bypassCtx).Channel.Query().Where(channel.ID(channelID)).Only(bypassCtx)
	})
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to load channel %d for share/use settlement: %w", channelID, err)
	}

	if loaded == nil || loaded.Settings == nil || loaded.Settings.Share == nil {
		return 0, false, nil
	}

	share := loaded.Settings.Share
	if share.VisibilityOrDefault() != objects.ChannelVisibilityShared {
		return 0, false, nil
	}
	if share.OwnerUserID == nil || share.OwnerUserID.Type != ent.TypeUser || share.OwnerUserID.ID <= 0 {
		return 0, false, nil
	}

	return share.OwnerUserID.ID, true, nil
}

func shareUseLockedUserIDs(callerUserID, ownerUserID int) []int {
	userIDs := []int{callerUserID, ownerUserID}
	sort.Ints(userIDs)
	return userIDs
}

func shareUseDebitIdempotencyKey(usageLogID, callerUserID int) string {
	return fmt.Sprintf("share_use:usage_log:%d:user:%d:debit", usageLogID, callerUserID)
}

func shareUseCreditIdempotencyKey(usageLogID, ownerUserID int) string {
	return fmt.Sprintf("share_use:usage_log:%d:user:%d:credit", usageLogID, ownerUserID)
}

func userPointLedgerEntryExists(ctx context.Context, tx *sql.Tx, dialectName, idempotencyKey string) (bool, error) {
	query := fmt.Sprintf("SELECT id FROM user_point_ledger_entries WHERE idempotency_key = %s", relayPlaceholder(dialectName, 1))
	var id int
	err := tx.QueryRowContext(ctx, query, idempotencyKey).Scan(&id)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, fmt.Errorf("failed to check share/use settlement idempotency: %w", err)
}

func ensureUserPointAccountExists(ctx context.Context, tx *sql.Tx, dialectName string, userID int) error {
	if userID <= 0 {
		return fmt.Errorf("user id must be greater than 0 for point account")
	}

	switch dialectName {
	case dialect.MySQL:
		query := `INSERT INTO user_point_accounts (user_id, available_points, pending_points, frozen_points, lifetime_earned, lifetime_spent, version, created_at, updated_at)
VALUES (?, '0', '0', '0', '0', '0', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE updated_at = updated_at`
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return fmt.Errorf("failed to ensure point account exists: %w", err)
		}
	case dialect.Postgres:
		query := `INSERT INTO user_point_accounts (user_id, available_points, pending_points, frozen_points, lifetime_earned, lifetime_spent, version, created_at, updated_at)
VALUES ($1, '0', '0', '0', '0', '0', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(user_id) DO NOTHING`
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return fmt.Errorf("failed to ensure point account exists: %w", err)
		}
	default:
		query := `INSERT OR IGNORE INTO user_point_accounts (user_id, available_points, pending_points, frozen_points, lifetime_earned, lifetime_spent, version, created_at, updated_at)
VALUES (?, '0', '0', '0', '0', '0', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return fmt.Errorf("failed to ensure point account exists: %w", err)
		}
	}

	return nil
}

func loadUserPointAccountForUpdate(ctx context.Context, tx *sql.Tx, dialectName string, userID int) (*userPointAccountSnapshot, error) {
	if err := ensureUserPointAccountExists(ctx, tx, dialectName, userID); err != nil {
		return nil, err
	}

	if dialectName == dialect.SQLite {
		query := fmt.Sprintf("UPDATE user_point_accounts SET updated_at = updated_at WHERE user_id = %s", relayPlaceholder(dialectName, 1))
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return nil, fmt.Errorf("failed to lock point account for settlement: %w", err)
		}
	}

	lockClause := ""
	if dialectName != dialect.SQLite {
		lockClause = " FOR UPDATE"
	}
	query := fmt.Sprintf("SELECT available_points, pending_points, frozen_points, lifetime_earned, lifetime_spent, version FROM user_point_accounts WHERE user_id = %s%s", relayPlaceholder(dialectName, 1), lockClause)

	var (
		availableRaw sql.NullString
		pendingRaw   sql.NullString
		frozenRaw    sql.NullString
		earnedRaw    sql.NullString
		spentRaw     sql.NullString
		version      int64
	)
	if err := tx.QueryRowContext(ctx, query, userID).Scan(&availableRaw, &pendingRaw, &frozenRaw, &earnedRaw, &spentRaw, &version); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("point account %d not found for settlement", userID)
		}
		return nil, fmt.Errorf("failed to load point account %d: %w", userID, err)
	}

	availablePoints, err := parseRelayDecimal(availableRaw, "point account available balance")
	if err != nil {
		return nil, err
	}
	pendingPoints, err := parseRelayDecimal(pendingRaw, "point account pending balance")
	if err != nil {
		return nil, err
	}
	frozenPoints, err := parseRelayDecimal(frozenRaw, "point account frozen balance")
	if err != nil {
		return nil, err
	}
	lifetimeEarned, err := parseRelayDecimal(earnedRaw, "point account lifetime earned")
	if err != nil {
		return nil, err
	}
	lifetimeSpent, err := parseRelayDecimal(spentRaw, "point account lifetime spent")
	if err != nil {
		return nil, err
	}

	return &userPointAccountSnapshot{
		UserID:          userID,
		AvailablePoints: availablePoints,
		PendingPoints:   pendingPoints,
		FrozenPoints:    frozenPoints,
		LifetimeEarned:  lifetimeEarned,
		LifetimeSpent:   lifetimeSpent,
		Version:         version,
	}, nil
}

func updateUserPointAccount(ctx context.Context, tx *sql.Tx, dialectName string, account *userPointAccountSnapshot) error {
	if account == nil || account.UserID <= 0 {
		return fmt.Errorf("point account snapshot is invalid")
	}

	query := fmt.Sprintf(
		"UPDATE user_point_accounts SET available_points = %s, pending_points = %s, frozen_points = %s, lifetime_earned = %s, lifetime_spent = %s, version = version + 1, updated_at = %s WHERE user_id = %s",
		relayPlaceholder(dialectName, 1),
		relayPlaceholder(dialectName, 2),
		relayPlaceholder(dialectName, 3),
		relayPlaceholder(dialectName, 4),
		relayPlaceholder(dialectName, 5),
		relayPlaceholder(dialectName, 6),
		relayPlaceholder(dialectName, 7),
	)
	_, err := tx.ExecContext(
		ctx,
		query,
		account.AvailablePoints.String(),
		account.PendingPoints.String(),
		account.FrozenPoints.String(),
		account.LifetimeEarned.String(),
		account.LifetimeSpent.String(),
		time.Now().UTC(),
		account.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update point account %d: %w", account.UserID, err)
	}

	return nil
}

func insertUserPointLedgerEntry(ctx context.Context, tx *sql.Tx, dialectName string, input userPointLedgerInsertInput) error {
	columns := []string{
		"user_id",
		"direction",
		"scene",
		"points",
		"balance_before",
		"balance_after",
		"idempotency_key",
		"related_channel_id",
		"related_request_id",
		"related_usage_log_id",
		"related_api_key_id",
		"related_project_id",
		"conversion_rate_snapshot",
		"settlement_status",
		"remark",
	}
	placeholders := relayPlaceholders(dialectName, len(columns), 1)
	query := fmt.Sprintf("INSERT INTO user_point_ledger_entries (%s) VALUES (%s)", joinStrings(columns), joinStrings(placeholders))

	_, err := tx.ExecContext(
		ctx,
		query,
		input.UserID,
		input.Direction,
		input.Scene,
		input.Points.String(),
		input.BalanceBefore.String(),
		input.BalanceAfter.String(),
		input.IdempotencyKey,
		nullableInt(input.RelatedChannelID),
		nullableInt(input.RelatedRequestID),
		nullableInt(input.RelatedUsageLogID),
		nullableInt(input.RelatedAPIKeyID),
		nullableInt(input.RelatedProjectID),
		nullableString(input.ConversionRateSnapshot),
		input.SettlementStatus,
		nullableString(input.Remark),
	)
	if err != nil {
		return fmt.Errorf("failed to insert user point ledger entry: %w", err)
	}

	return nil
}

func joinStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	joined := values[0]
	for i := 1; i < len(values); i++ {
		joined += "," + values[i]
	}
	return joined
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func logShareUseSettlementSkip(ctx context.Context, usageLog *ent.UsageLog, reason string) {
	if usageLog == nil || !log.DebugEnabled(ctx) {
		return
	}
	log.Debug(ctx, "share/use settlement skipped", log.Int("usage_log_id", usageLog.ID), log.String("reason", reason))
}
