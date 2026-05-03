package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
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
	ledgerEntries, err := h.ShareUseWalletService.ListLedgerEntries(c.Request.Context(), user.ID)
	if err != nil {
		JSONError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ledgerEntries": ledgerEntries, "data": ledgerEntries})
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
