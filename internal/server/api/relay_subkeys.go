package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
)

type RelaySubKeyHandlersParams struct {
	fx.In

	RelayAdminService *biz.RelayAdminService
}

type RelaySubKeyHandlers struct {
	RelayAdminService *biz.RelayAdminService
}

type RelaySubKeyEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func NewRelaySubKeyHandlers(params RelaySubKeyHandlersParams) *RelaySubKeyHandlers {
	return &RelaySubKeyHandlers{RelayAdminService: params.RelayAdminService}
}

func RelaySubKeyRESTContract() []RelaySubKeyEndpoint {
	return []RelaySubKeyEndpoint{
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/products"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/products/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/products"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/products/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/product-channels"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/keys"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/keys/:id/status"},
		{Method: http.MethodPatch, Path: "/admin/relay-subkeys/keys/:id/limits"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id/wallet"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/keys/:id/ledger"},
		{Method: http.MethodPost, Path: "/admin/relay-subkeys/wallets/recharge"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/requests"},
		{Method: http.MethodGet, Path: "/admin/relay-subkeys/channel-pool-health"},
		{Method: http.MethodGet, Path: "/admin/projects/:projectId/relay-subkeys/overview"},
		{Method: http.MethodGet, Path: "/admin/projects/:projectId/relay-subkeys/usage"},
	}
}

func (h *RelaySubKeyHandlers) ListProducts(c *gin.Context) {
	products, err := h.RelayAdminService.ListProducts(c.Request.Context())
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"products": products, "data": products})
}

func (h *RelaySubKeyHandlers) GetProduct(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	product, err := h.RelayAdminService.GetProduct(c.Request.Context(), id)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product, "data": product})
}

func (h *RelaySubKeyHandlers) CreateProduct(c *gin.Context) {
	var input biz.RelayAdminProductCreateInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	product, err := h.RelayAdminService.CreateProduct(c.Request.Context(), input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": product, "data": product})
}

func (h *RelaySubKeyHandlers) UpdateProduct(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	var input biz.RelayAdminProductUpdateInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	product, err := h.RelayAdminService.UpdateProduct(c.Request.Context(), id, input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": product, "data": product})
}

func (h *RelaySubKeyHandlers) CreateProductChannel(c *gin.Context) {
	var input biz.RelayAdminChannelBindingInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	channel, err := h.RelayAdminService.CreateProductChannelBinding(c.Request.Context(), input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"channel": channel, "data": channel})
}

func (h *RelaySubKeyHandlers) ListKeys(c *gin.Context) {
	projectID, ok := relaySubKeyOptionalIDQuery(c, "projectId")
	if !ok {
		return
	}
	keys, err := h.RelayAdminService.ListKeys(c.Request.Context(), projectID)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys, "data": keys})
}

func (h *RelaySubKeyHandlers) GetKey(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	key, err := h.RelayAdminService.GetKey(c.Request.Context(), id)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "data": key})
}

func (h *RelaySubKeyHandlers) CreateKey(c *gin.Context) {
	var input biz.RelayAdminCreateKeyInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	key, err := h.RelayAdminService.CreateKey(c.Request.Context(), input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"key": key, "data": key})
}

func (h *RelaySubKeyHandlers) UpdateKeyStatus(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	var input biz.RelayAdminKeyStatusInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	key, err := h.RelayAdminService.UpdateKeyStatus(c.Request.Context(), id, input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "data": key})
}

func (h *RelaySubKeyHandlers) UpdateKeyLimits(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	var input biz.RelayAdminKeyLimitsInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	key, err := h.RelayAdminService.UpdateKeyLimits(c.Request.Context(), id, input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "data": key})
}

func (h *RelaySubKeyHandlers) GetWallet(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	wallet, err := h.RelayAdminService.GetWallet(c.Request.Context(), id)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallet, "data": wallet})
}

func (h *RelaySubKeyHandlers) ListLedger(c *gin.Context) {
	id, ok := relaySubKeyIDParam(c, "id")
	if !ok {
		return
	}
	ledger, err := h.RelayAdminService.ListLedgerEntries(c.Request.Context(), id)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ledgerEntries": ledger, "data": ledger})
}

func (h *RelaySubKeyHandlers) RechargeWallet(c *gin.Context) {
	var input biz.RelayAdminRechargeInput
	if !relaySubKeyBindJSON(c, &input) {
		return
	}
	entry, err := h.RelayAdminService.RechargeWallet(c.Request.Context(), input)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ledgerEntry": entry, "data": entry})
}

func (h *RelaySubKeyHandlers) ListRequests(c *gin.Context) {
	projectID, ok := relaySubKeyOptionalIDQuery(c, "projectId")
	if !ok {
		return
	}
	requests, err := h.RelayAdminService.ListRequestTraces(c.Request.Context(), projectID)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests, "data": requests})
}

func (h *RelaySubKeyHandlers) ChannelPoolHealth(c *gin.Context) {
	health, err := h.RelayAdminService.ListChannelPoolHealth(c.Request.Context())
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"channelPoolHealth": health, "data": health})
}

func (h *RelaySubKeyHandlers) ProjectOverview(c *gin.Context) {
	projectID, ok := relaySubKeyIDParam(c, "projectId")
	if !ok {
		return
	}
	overview, err := h.RelayAdminService.GetOverview(c.Request.Context(), &projectID)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"overview": overview, "data": overview})
}

func (h *RelaySubKeyHandlers) ProjectUsage(c *gin.Context) {
	projectID, ok := relaySubKeyIDParam(c, "projectId")
	if !ok {
		return
	}
	usage, err := h.RelayAdminService.GetUsage(c.Request.Context(), &projectID)
	if err != nil {
		relaySubKeyJSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"usage": usage, "data": usage})
}

func relaySubKeyBindJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		JSONError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return false
	}
	return true
}

func relaySubKeyIDParam(c *gin.Context, name string) (int, bool) {
	id, err := relaySubKeyParseID(c.Param(name), name)
	if err != nil {
		JSONError(c, http.StatusBadRequest, err)
		return 0, false
	}
	return id, true
}

func relaySubKeyOptionalIDQuery(c *gin.Context, name string) (*int, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, true
	}
	id, err := relaySubKeyParseID(value, name)
	if err != nil {
		JSONError(c, http.StatusBadRequest, err)
		return nil, false
	}
	return &id, true
}

func relaySubKeyParseID(value, name string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	if guid, err := objects.ParseGUID(value); err == nil {
		if guid.ID <= 0 {
			return 0, fmt.Errorf("%s must be greater than 0", name)
		}
		return guid.ID, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a numeric id or AxonHub GUID", name)
	}
	return id, nil
}

func relaySubKeyJSONError(c *gin.Context, err error) {
	message := strings.ToLower(err.Error())
	status := http.StatusInternalServerError
	if strings.Contains(message, "not found") {
		status = http.StatusNotFound
	} else if strings.Contains(message, "required") || strings.Contains(message, "invalid") || strings.Contains(message, "unsupported") || strings.Contains(message, "cannot") || strings.Contains(message, "must") || strings.Contains(message, "negative") || strings.Contains(message, "already exists") || strings.Contains(message, "already bound") {
		status = http.StatusBadRequest
	}
	JSONError(c, status, err)
}
