package biz

import (
	"context"
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/userpointaccount"
	"github.com/looplj/axonhub/internal/ent/userpointledgerentry"
)

type ShareUseWalletServiceParams struct {
	fx.In

	Ent *ent.Client
}

type ShareUseWalletService struct {
	*AbstractService
}

type ShareUsePointWalletView struct {
	ID              string  `json:"id"`
	UserID          string  `json:"userId"`
	AvailablePoints float64 `json:"availablePoints"`
	PendingPoints   float64 `json:"pendingPoints"`
	FrozenPoints    float64 `json:"frozenPoints"`
	LifetimeEarned  float64 `json:"lifetimeEarned"`
	LifetimeSpent   float64 `json:"lifetimeSpent"`
	UpdatedAt       string  `json:"updatedAt"`
}

type ShareUsePointLedgerEntryView struct {
	ID                     string  `json:"id"`
	UserID                 string  `json:"userId"`
	Direction              string  `json:"direction"`
	Scene                  string  `json:"scene"`
	Points                 float64 `json:"points"`
	BalanceBefore          float64 `json:"balanceBefore"`
	BalanceAfter           float64 `json:"balanceAfter"`
	RelatedChannelID       string  `json:"relatedChannelId,omitempty"`
	RelatedRequestID       string  `json:"relatedRequestId,omitempty"`
	RelatedUsageLogID      string  `json:"relatedUsageLogId,omitempty"`
	RelatedAPIKeyID        string  `json:"relatedApiKeyId,omitempty"`
	RelatedProjectID       string  `json:"relatedProjectId,omitempty"`
	ConversionRateSnapshot string  `json:"conversionRateSnapshot,omitempty"`
	SettlementStatus       string  `json:"settlementStatus"`
	Remark                 string  `json:"remark,omitempty"`
	CreatedAt              string  `json:"createdAt"`
}

type ShareUseWalletUsageView struct {
	Wallet        *ShareUsePointWalletView       `json:"wallet"`
	LedgerEntries []ShareUsePointLedgerEntryView `json:"ledgerEntries"`
}

func NewShareUseWalletService(params ShareUseWalletServiceParams) *ShareUseWalletService {
	return &ShareUseWalletService{AbstractService: &AbstractService{db: params.Ent}}
}

func (s *ShareUseWalletService) GetWallet(ctx context.Context, userID int) (*ShareUsePointWalletView, error) {
	wallets, err := s.listWallets(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(wallets) == 0 {
		return nil, nil
	}
	return &wallets[0], nil
}

func (s *ShareUseWalletService) ListLedgerEntries(ctx context.Context, userID int) ([]ShareUsePointLedgerEntryView, error) {
	return s.listLedgerEntries(ctx, userID)
}

func (s *ShareUseWalletService) GetUsage(ctx context.Context, userID int) (*ShareUseWalletUsageView, error) {
	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	ledgerEntries, err := s.ListLedgerEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &ShareUseWalletUsageView{Wallet: wallet, LedgerEntries: ledgerEntries}, nil
}

func (s *ShareUseWalletService) listWallets(ctx context.Context, userID int) ([]ShareUsePointWalletView, error) {
	rows, err := s.entFromContext(ctx).UserPointAccount.Query().
		Where(userpointaccount.UserID(userID)).
		Order(userpointaccount.ByID(entsql.OrderAsc())).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list share/use point wallets: %w", err)
	}

	out := make([]ShareUsePointWalletView, 0, len(rows))
	for _, row := range rows {
		out = append(out, shareUseWalletFromEnt(row))
	}
	return out, nil
}

func (s *ShareUseWalletService) listLedgerEntries(ctx context.Context, userID int) ([]ShareUsePointLedgerEntryView, error) {
	rows, err := s.entFromContext(ctx).UserPointLedgerEntry.Query().
		Where(userpointledgerentry.UserID(userID)).
		Order(
			userpointledgerentry.ByCreatedAt(entsql.OrderDesc()),
			userpointledgerentry.ByID(entsql.OrderDesc()),
		).
		Limit(200).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list share/use point ledger entries: %w", err)
	}

	out := make([]ShareUsePointLedgerEntryView, 0, len(rows))
	for _, row := range rows {
		out = append(out, shareUseLedgerEntryFromEnt(row))
	}
	return out, nil
}

func shareUseWalletFromEnt(row *ent.UserPointAccount) ShareUsePointWalletView {
	return ShareUsePointWalletView{
		ID:              relayStringID(row.ID),
		UserID:          relayStringID(row.UserID),
		AvailablePoints: shareUseDecimalFloat(row.AvailablePoints),
		PendingPoints:   shareUseDecimalFloat(row.PendingPoints),
		FrozenPoints:    shareUseDecimalFloat(row.FrozenPoints),
		LifetimeEarned:  shareUseDecimalFloat(row.LifetimeEarned),
		LifetimeSpent:   shareUseDecimalFloat(row.LifetimeSpent),
		UpdatedAt:       row.UpdatedAt.Format(timeRFC3339),
	}
}

func shareUseLedgerEntryFromEnt(row *ent.UserPointLedgerEntry) ShareUsePointLedgerEntryView {
	view := ShareUsePointLedgerEntryView{
		ID:               relayStringID(row.ID),
		UserID:           relayStringID(row.UserID),
		Direction:        string(row.Direction),
		Scene:            string(row.Scene),
		Points:           shareUseDecimalFloat(row.Points),
		BalanceBefore:    shareUseDecimalFloat(row.BalanceBefore),
		BalanceAfter:     shareUseDecimalFloat(row.BalanceAfter),
		SettlementStatus: string(row.SettlementStatus),
		CreatedAt:        row.CreatedAt.Format(timeRFC3339),
	}
	if row.RelatedChannelID != nil {
		view.RelatedChannelID = relayStringID(*row.RelatedChannelID)
	}
	if row.RelatedRequestID != nil {
		view.RelatedRequestID = relayStringID(*row.RelatedRequestID)
	}
	if row.RelatedUsageLogID != nil {
		view.RelatedUsageLogID = relayStringID(*row.RelatedUsageLogID)
	}
	if row.RelatedAPIKeyID != nil {
		view.RelatedAPIKeyID = relayStringID(*row.RelatedAPIKeyID)
	}
	if row.RelatedProjectID != nil {
		view.RelatedProjectID = relayStringID(*row.RelatedProjectID)
	}
	if row.ConversionRateSnapshot != nil {
		view.ConversionRateSnapshot = strings.TrimSpace(*row.ConversionRateSnapshot)
	}
	if row.Remark != nil {
		view.Remark = strings.TrimSpace(*row.Remark)
	}
	return view
}

func shareUseDecimalFloat(raw string) float64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return 0
	}
	return parsed.InexactFloat64()
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
