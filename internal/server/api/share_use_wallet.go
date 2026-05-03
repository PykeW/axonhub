package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/userpointledgerentry"
	"github.com/looplj/axonhub/internal/server/biz"
)

type ShareUseWalletHandlersParams struct {
	fx.In

	ShareUseWalletService *biz.ShareUseWalletService
}

type ShareUseWalletHandlers struct {
	ShareUseWalletService *biz.ShareUseWalletService
}

func NewShareUseWalletHandlers(params ShareUseWalletHandlersParams) *ShareUseWalletHandlers {
	return &ShareUseWalletHandlers{ShareUseWalletService: params.ShareUseWalletService}
}

func (h *ShareUseWalletHandlers) GetWallet(c *gin.Context) {
	user, exists := currentShareUseWalletUser(c)
	if !exists {
		return
	}
	wallet, err := h.ShareUseWalletService.GetWallet(c.Request.Context(), user.ID)
	if err != nil {
		JSONError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallet, "data": wallet})
}

func (h *ShareUseWalletHandlers) ListLedger(c *gin.Context) {
	user, exists := currentShareUseWalletUser(c)
	if !exists {
		return
	}
	page := parseShareUseLedgerInt(c, "page", 0)
	pageSize := parseShareUseLedgerInt(c, "pageSize", 20)
	filters, err := parseShareUseLedgerFilters(c)
	if err != nil {
		JSONError(c, http.StatusBadRequest, err)
		return
	}
	ledgerPage, err := h.ShareUseWalletService.ListLedgerPage(c.Request.Context(), user.ID, page, pageSize, filters)
	if err != nil {
		JSONError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"entries":       ledgerPage.Entries,
		"ledgerEntries": ledgerPage.Entries,
		"totalCount":    ledgerPage.TotalCount,
		"page":          ledgerPage.Page,
		"pageSize":      ledgerPage.PageSize,
		"hasNext":       ledgerPage.HasNext,
		"hasPrev":       ledgerPage.HasPrev,
		"data":          ledgerPage,
	})
}

func (h *ShareUseWalletHandlers) GetUsage(c *gin.Context) {
	user, exists := currentShareUseWalletUser(c)
	if !exists {
		return
	}
	usage, err := h.ShareUseWalletService.GetUsage(c.Request.Context(), user.ID)
	if err != nil {
		JSONError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": usage, "data": usage})
}

func currentShareUseWalletUser(c *gin.Context) (*ent.User, bool) {
	user, exists := contexts.GetUser(c.Request.Context())
	if !exists || user == nil || user.ID <= 0 {
		JSONError(c, http.StatusUnauthorized, fmt.Errorf("current user context is required"))
		return nil, false
	}
	return user, true
}

func parseShareUseLedgerInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseShareUseLedgerFilters(c *gin.Context) (biz.ShareUseLedgerFilters, error) {
	filters := biz.ShareUseLedgerFilters{}

	scene, err := parseShareUseLedgerScene(c.Query("scene"))
	if err != nil {
		return biz.ShareUseLedgerFilters{}, err
	}
	filters.Scene = scene

	direction, err := parseShareUseLedgerDirection(c.Query("direction"))
	if err != nil {
		return biz.ShareUseLedgerFilters{}, err
	}
	filters.Direction = direction

	createdAtGTE, err := parseShareUseLedgerTime(c.Query("createdAtGTE"), "createdAtGTE")
	if err != nil {
		return biz.ShareUseLedgerFilters{}, err
	}
	filters.CreatedAtGTE = createdAtGTE

	createdAtLTE, err := parseShareUseLedgerTime(c.Query("createdAtLTE"), "createdAtLTE")
	if err != nil {
		return biz.ShareUseLedgerFilters{}, err
	}
	filters.CreatedAtLTE = createdAtLTE

	if filters.CreatedAtGTE != nil && filters.CreatedAtLTE != nil && filters.CreatedAtGTE.After(*filters.CreatedAtLTE) {
		return biz.ShareUseLedgerFilters{}, fmt.Errorf("createdAtGTE must be before or equal to createdAtLTE")
	}

	return filters, nil
}

func parseShareUseLedgerScene(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if err := userpointledgerentry.SceneValidator(userpointledgerentry.Scene(value)); err != nil {
		return "", fmt.Errorf("invalid scene %q: must be one of %s, %s, %s, %s", value, userpointledgerentry.SceneContributionPending, userpointledgerentry.SceneContributionReward, userpointledgerentry.SceneConsume, userpointledgerentry.SceneAdjustment)
	}
	return value, nil
}

func parseShareUseLedgerDirection(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if err := userpointledgerentry.DirectionValidator(userpointledgerentry.Direction(value)); err != nil {
		return "", fmt.Errorf("invalid direction %q: must be one of %s or %s", value, userpointledgerentry.DirectionCredit, userpointledgerentry.DirectionDebit)
	}
	return value, nil
}

func parseShareUseLedgerTime(raw, key string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be an RFC3339 timestamp", key)
	}
	return &parsed, nil
}
