package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/enttest"
	"github.com/looplj/axonhub/internal/ent/userpointledgerentry"
	"github.com/looplj/axonhub/internal/server/biz"
)

type shareUseLedgerResponse struct {
	Entries       []shareUseLedgerResponseEntry `json:"entries"`
	LedgerEntries []shareUseLedgerResponseEntry `json:"ledgerEntries"`
	TotalCount    int                           `json:"totalCount"`
	Page          int                           `json:"page"`
	PageSize      int                           `json:"pageSize"`
	HasNext       bool                          `json:"hasNext"`
	HasPrev       bool                          `json:"hasPrev"`
}

type shareUseLedgerResponseEntry struct {
	Remark string `json:"remark"`
}

func TestShareUseWalletHandlers_ListLedgerReturnsPaginatedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = 101
	client := newShareUseWalletTestClient(t)
	seedShareUseWalletTestData(t, client, userID)

	handlers := NewShareUseWalletHandlers(ShareUseWalletHandlersParams{
		ShareUseWalletService: biz.NewShareUseWalletService(biz.ShareUseWalletServiceParams{Ent: client}),
	})

	router := newShareUseWalletTestRouter(client, userID, func(r *gin.Engine) {
		r.GET("/admin/share-use/ledger", handlers.ListLedger)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/share-use/ledger?page=1&pageSize=2", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload shareUseLedgerResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 1, len(payload.Entries))
	require.Equal(t, 1, len(payload.LedgerEntries))
	require.Equal(t, "first", payload.Entries[0].Remark)
	require.Equal(t, "first", payload.LedgerEntries[0].Remark)
	require.Equal(t, 3, payload.TotalCount)
	require.Equal(t, 1, payload.Page)
	require.Equal(t, 2, payload.PageSize)
	require.False(t, payload.HasNext)
	require.True(t, payload.HasPrev)
}

func TestShareUseWalletHandlers_ListLedgerSupportsFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = 102
	client := newShareUseWalletTestClient(t)
	seedShareUseWalletTestData(t, client, userID)

	handlers := NewShareUseWalletHandlers(ShareUseWalletHandlersParams{
		ShareUseWalletService: biz.NewShareUseWalletService(biz.ShareUseWalletServiceParams{Ent: client}),
	})

	router := newShareUseWalletTestRouter(client, userID, func(r *gin.Engine) {
		r.GET("/admin/share-use/ledger", handlers.ListLedger)
	})

	base := shareUseWalletTestBaseTime()
	query := url.Values{
		"scene":        []string{string(userpointledgerentry.SceneConsume)},
		"direction":    []string{string(userpointledgerentry.DirectionDebit)},
		"createdAtGTE": []string{base.Add(30 * time.Second).Format(time.RFC3339)},
		"createdAtLTE": []string{base.Add(90 * time.Second).Format(time.RFC3339)},
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/share-use/ledger?"+query.Encode(), nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload shareUseLedgerResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, []shareUseLedgerResponseEntry{{Remark: "second"}}, payload.Entries)
	require.Equal(t, []shareUseLedgerResponseEntry{{Remark: "second"}}, payload.LedgerEntries)
	require.Equal(t, 1, payload.TotalCount)
	require.False(t, payload.HasNext)
	require.False(t, payload.HasPrev)
}

func TestRelaySubKeyHandlers_ListShareUseLedgerMatchesPaginatedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = 202
	client := newShareUseWalletTestClient(t)
	seedShareUseWalletTestData(t, client, userID)

	handlers := NewRelaySubKeyHandlers(RelaySubKeyHandlersParams{
		ShareUseWalletService: biz.NewShareUseWalletService(biz.ShareUseWalletServiceParams{Ent: client}),
	})

	router := newShareUseWalletTestRouter(client, userID, func(r *gin.Engine) {
		r.GET("/admin/share-use/ledger", handlers.ListShareUseLedger)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/share-use/ledger?page=0&pageSize=2", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload shareUseLedgerResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 2, len(payload.Entries))
	require.Equal(t, 2, len(payload.LedgerEntries))
	require.Equal(t, []shareUseLedgerResponseEntry{{Remark: "third"}, {Remark: "second"}}, payload.Entries)
	require.Equal(t, []shareUseLedgerResponseEntry{{Remark: "third"}, {Remark: "second"}}, payload.LedgerEntries)
	require.Equal(t, 3, payload.TotalCount)
	require.Equal(t, 0, payload.Page)
	require.Equal(t, 2, payload.PageSize)
	require.True(t, payload.HasNext)
	require.False(t, payload.HasPrev)
}

func TestRelaySubKeyHandlers_ListShareUseLedgerRejectsInvalidFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = 203
	client := newShareUseWalletTestClient(t)
	seedShareUseWalletTestData(t, client, userID)

	handlers := NewRelaySubKeyHandlers(RelaySubKeyHandlersParams{
		ShareUseWalletService: biz.NewShareUseWalletService(biz.ShareUseWalletServiceParams{Ent: client}),
	})

	router := newShareUseWalletTestRouter(client, userID, func(r *gin.Engine) {
		r.GET("/admin/share-use/ledger", handlers.ListShareUseLedger)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/share-use/ledger?scene=invalid", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func newShareUseWalletTestClient(t *testing.T) *ent.Client {
	t.Helper()

	client := enttest.NewEntClient(t, dialect.SQLite, fmt.Sprintf("file:share_use_wallet_%d?mode=memory&cache=shared&_fk=0", time.Now().UnixNano()))
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	return client
}

func newShareUseWalletTestRouter(client *ent.Client, userID int, register func(*gin.Engine)) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := ent.NewContext(authz.WithTestBypass(c.Request.Context()), client)
		ctx = contexts.WithUser(ctx, &ent.User{ID: userID, IsOwner: true})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	register(router)
	return router
}

func seedShareUseWalletTestData(t *testing.T, client *ent.Client, userID int) {
	t.Helper()

	ctx := ent.NewContext(authz.WithTestBypass(context.Background()), client)
	base := shareUseWalletTestBaseTime()

	_, err := client.UserPointAccount.Create().
		SetUserID(userID).
		SetAvailablePoints("6").
		SetLifetimeEarned("6").
		Save(ctx)
	require.NoError(t, err)

	entries := []struct {
		remark        string
		direction     userpointledgerentry.Direction
		scene         userpointledgerentry.Scene
		points        string
		balanceBefore string
		balanceAfter  string
		createdAt     time.Time
	}{
		{remark: "first", direction: userpointledgerentry.DirectionCredit, scene: userpointledgerentry.SceneContributionReward, points: "1", balanceBefore: "0", balanceAfter: "1", createdAt: base},
		{remark: "second", direction: userpointledgerentry.DirectionDebit, scene: userpointledgerentry.SceneConsume, points: "2", balanceBefore: "1", balanceAfter: "3", createdAt: base.Add(time.Minute)},
		{remark: "third", direction: userpointledgerentry.DirectionCredit, scene: userpointledgerentry.SceneAdjustment, points: "3", balanceBefore: "3", balanceAfter: "6", createdAt: base.Add(2 * time.Minute)},
	}

	for index, entry := range entries {
		_, err := client.UserPointLedgerEntry.Create().
			SetUserID(userID).
			SetDirection(entry.direction).
			SetScene(entry.scene).
			SetPoints(entry.points).
			SetBalanceBefore(entry.balanceBefore).
			SetBalanceAfter(entry.balanceAfter).
			SetIdempotencyKey(fmt.Sprintf("share-use-test-%d-%d", userID, index)).
			SetRemark(entry.remark).
			SetCreatedAt(entry.createdAt).
			SetUpdatedAt(entry.createdAt).
			Save(ctx)
		require.NoError(t, err)
	}
}

func shareUseWalletTestBaseTime() time.Time {
	return time.Date(2026, time.May, 3, 12, 0, 0, 0, time.UTC)
}
