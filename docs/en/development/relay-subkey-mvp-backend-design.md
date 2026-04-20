# Self-Hosted Relay + Shared-Capacity Sub-Key MVP: Backend and Data Design

## Background

AxonHub already has most of the primitives required for a self-hosted relay MVP:

- `channels` already carry upstream provider credentials, model capabilities, routing settings, and health state.
- `api_keys` already handle downstream API authentication and naturally inherit `project_id` isolation.
- `requests`, `request_executions`, and `usage_logs` already cover request facts, upstream execution facts, and token/cost facts.
- `provider_quota_status` already gives us a quota polling hook for providers such as Claude Code and Codex.

This MVP does not resell upstream keys directly. Instead, the operator runs one AxonHub relay and issues its own downstream sub-keys. Multiple sub-keys share one upstream capacity pool, while balance, quotas, suspension state, and risk controls remain independent per sub-key.

## MVP Scope

1. Self-hosted relay only; no decentralized matching and no multi-seller marketplace.
2. The operator manually configures products, upstream channels, sub-keys, recharges, and freezes.
3. Downstream traffic keeps using existing OpenAI / Anthropic / Codex compatible endpoints; no new protocol branch is introduced.
4. Billing starts with a prepaid balance plus hard quotas model; payment gateway integration, invoicing, and reseller revenue sharing are deferred.
5. Shared capacity is implemented as "one product bound to many upstream channels", not by exposing a provider key directly to end users.

## Design Principles

- **Reuse the request pipeline first**: `requests`, `request_executions`, and `usage_logs` remain the primary request facts.
- **Separate auth from relay business logic**: downstream auth keeps living in `api_keys`; relay-specific state moves to dedicated tables.
- **Keep accounting immutable**: all balance changes must go through an append-only ledger.
- **Route by pool, not by single key**: a sold sub-key binds to a product, and the product binds to a shared upstream channel pool.
- **Keep the synchronous path small**: the request path should only do status, balance, and hard-limit checks; heavier analytics should use summary tables.

## Existing Tables to Reuse

| Table | Role in the MVP | Treatment |
| --- | --- | --- |
| `projects` | Buyer tenant isolation and console scope | Keep as-is |
| `users` | Admin and operator accounts | Keep as-is |
| `api_keys` | Customer-facing authentication credentials | Keep as-is; bind business semantics through `relay_keys.api_key_id` |
| `channels` | Upstream provider accounts / sub-keys / OAuth channels | Keep as-is; continue to carry the real upstream credentials |
| `provider_quota_status` | Upstream quota polling state | Keep as-is; use as a routing filter |
| `requests` | Downstream request fact table | Keep as-is; use as the settlement anchor |
| `request_executions` | Actual upstream channel execution facts | Keep as-is; use as the routing fact table |
| `usage_logs` | Actual token and upstream cost facts | Keep as-is; use as the upstream cost truth source |
| `channel_model_prices` / `channel_model_price_versions` | Upstream pricing baseline | Keep as-is |
| `system` | Relay feature flags and default policies | Add config items if needed, but no separate new system table |

### Why not add an upstream account table first?

`channels` already act as the lifecycle container for one upstream account, one upstream sub-key, or one provider credential, and they are already integrated with request execution, quota polling, model pricing, and auto-disable logic. Reusing `channels` in the MVP avoids:

- maintaining a second credential lifecycle in parallel with `channels`;
- introducing a dual routing model into the request path;
- re-implementing quota, health, and fallback logic twice.

## New Table List

The MVP should add 6 tables, all managed by Ent under `internal/ent/schema/`.

### 1. `relay_products`

Suggested schema file: `internal/ent/schema/relay_product.go`

Purpose: define the sellable shared-capacity product exposed to downstream users.

Core fields:

- `id`
- `code`: unique product code used by the operator console
- `name`
- `provider_type`: e.g. `claudecode`, `codex`, `openai_compatible`
- `access_mode`: fixed to `shared_capacity`
- `billing_mode`: `prepaid` / `quota_only`
- `status`: `draft` / `active` / `archived`
- `currency`
- `list_price_config`: JSON describing downstream billing rules
- `allowed_models`: JSON array of allowed models at product level
- `request_timeout_seconds`
- `created_at` / `updated_at` / `deleted_at`

Recommended indexes:

- unique index on `code`
- index on `status`
- index on `provider_type`

Notes: the product is the sell-side aggregation object; it does not store upstream credentials directly.

### 2. `relay_product_channels`

Suggested schema file: `internal/ent/schema/relay_product_channel.go`

Purpose: bind one product to many upstream `channels`, forming the shared-capacity pool.

Core fields:

- `id`
- `product_id`
- `channel_id`
- `priority`: lower value means higher priority
- `weight`: weight inside the same priority tier
- `status`: `active` / `paused`
- `allow_fallback`
- `model_filter`: JSON or string rule to constrain which models this channel may serve for the product
- `max_inflight`
- `created_at` / `updated_at`

Recommended indexes:

- unique index on `(product_id, channel_id)`
- composite index on `(product_id, status, priority)`

Notes: shared capacity is fundamentally expressed as "product -> channel pool"; this table is the routing core.

### 3. `relay_keys`

Suggested schema file: `internal/ent/schema/relay_key.go`

Purpose: add relay business semantics on top of existing `api_keys`, representing the sold downstream sub-key.

Core fields:

- `id`
- `api_key_id`: unique binding to existing `api_keys`
- `project_id`
- `product_id`
- `owner_user_id`: nullable; supports operator-created keys or robot accounts
- `display_name`
- `status`: `active` / `suspended` / `exhausted` / `archived`
- `balance_mode`: `prepaid` / `quota_only`
- `daily_request_limit`
- `daily_token_limit`
- `monthly_cost_limit`
- `concurrency_limit`
- `expires_at`
- `last_used_at`
- `created_at` / `updated_at` / `deleted_at`

Recommended indexes:

- unique index on `api_key_id`
- composite index on `(project_id, status)`
- composite index on `(product_id, status)`
- index on `expires_at`

Notes:

- `api_keys` still own protocol-level authentication;
- `relay_keys` own balance, quotas, product binding, and risk-control state;
- this prevents `api_keys.profiles` from absorbing authentication, routing, and commercial billing at the same time.

### 4. `relay_wallets`

Suggested schema file: `internal/ent/schema/relay_wallet.go`

Purpose: store the balance snapshot for each relay key so the request path can perform a fast synchronous check.

Core fields:

- `id`
- `relay_key_id`: unique
- `currency`
- `available_amount`
- `frozen_amount`
- `overdraft_limit`
- `version`: optimistic lock version
- `updated_at`

Recommended indexes:

- unique index on `relay_key_id`

Notes:

- the wallet table stores only the current snapshot;
- the ledger table remains the accounting truth source;
- charging should use optimistic locking on `version` to prevent concurrent overdrafts.

### 5. `relay_wallet_ledger_entries`

Suggested schema file: `internal/ent/schema/relay_wallet_ledger_entry.go`

Purpose: record immutable accounting events such as recharge, consume, refund, freeze, unfreeze, and manual adjustment.

Core fields:

- `id`
- `relay_key_id`
- `request_id`: nullable; set on request consumption
- `usage_log_id`: nullable; links settlement to usage
- `direction`: `credit` / `debit`
- `scene`: `recharge` / `consume` / `refund` / `freeze` / `unfreeze` / `manual_adjust`
- `amount`
- `balance_before`
- `balance_after`
- `upstream_cost`: nullable; keeps the true upstream cost
- `price_snapshot`: JSON snapshot of the downstream pricing rule
- `idempotency_key`: unique, used to prevent duplicate charging
- `operator_user_id`: nullable; set on manual adjustments
- `remark`
- `created_at`

Recommended indexes:

- unique index on `idempotency_key`
- composite index on `(relay_key_id, created_at)`
- index on `request_id`
- index on `usage_log_id`

Notes: `usage-log:{id}:consume` is a good default idempotency key for a usage-based charge.

### 6. `relay_daily_usage_summaries`

Suggested schema file: `internal/ent/schema/relay_daily_usage_summary.go`

Purpose: provide lightweight aggregates for hard-limit checks and operator dashboards without scanning `usage_logs` on every request.

Core fields:

- `id`
- `relay_key_id`
- `stat_date`
- `request_count`
- `total_tokens`
- `total_charge`
- `total_upstream_cost`
- `last_request_id`
- `updated_at`

Recommended indexes:

- unique index on `(relay_key_id, stat_date)`
- index on `(stat_date, total_charge)`

Notes:

- day-level aggregation is sufficient for MVP daily quotas and basic dashboards;
- hour-level or model-level aggregates can be added later.

## Request and Settlement Flow

### 1. Authentication phase

1. Existing middleware still authenticates `X-API-Key` or Bearer keys through `api_keys`.
2. After auth succeeds, load `relay_keys` and `relay_wallets`.
3. Reject the request immediately if the relay key is not active, expired, over hard limit, or out of balance.

### 2. Routing phase

1. Resolve the product from `relay_keys.product_id`.
2. Load the candidate pool from `relay_product_channels`.
3. Filter the pool by:
   - `channels.status = enabled`
   - `relay_product_channels.status = active`
   - model match against `model_filter`
   - `provider_quota_status.ready = true` for providers that support quota polling
   - channel not currently auto-disabled
4. Pass the filtered candidate set to existing `ChannelService` health and priority logic.

### 3. Persistence phase

1. `RequestService.CreateRequest` still writes `requests`.
2. `RequestService.CreateRequestExecution` still writes `request_executions`.
3. The MVP should not introduce a second relay request fact table.

### 4. Settlement phase

1. `UsageLogService.CreateUsageLogFromRequest` writes `usage_logs` and captures the real token and upstream cost facts.
2. `RelaySettlementService` turns the usage log into an idempotent ledger debit.
3. After the debit succeeds, update the `relay_wallets` snapshot and upsert `relay_daily_usage_summaries`.
4. If the request fails and no `usage_logs` row is produced, charge nothing by default and leave reconciliation for a later repair flow.

### 5. Why not add a downstream charge field to `usage_logs`?

The MVP should keep upstream cost and downstream sell price separated:

- `usage_logs.total_cost` stays the upstream cost fact;
- `relay_wallet_ledger_entries.amount` stores what the customer is actually charged;
- this keeps the core usage fact table clean when promotions, monthly bundles, free quota, or manual refunds arrive later.

## Backend Module Decomposition

### Reused modules

| Module | Responsibility kept in the MVP |
| --- | --- |
| `APIKeyService` | Generate, query, and cache customer auth keys |
| `ChannelService` | Channel cache, health, fallback, and model capability checks |
| `RequestService` | Create `requests` and `request_executions` |
| `UsageLogService` | Persist real token usage and upstream cost |
| `QuotaService` | Existing API key quota logic remains a useful reference for relay hard limits |
| `ProviderQuotaService` | Poll upstream provider quota state |
| `ProjectService` | Buyer project management and isolation |

### New business modules

| Module | Suggested file | Responsibility | Primary dependencies |
| --- | --- | --- | --- |
| Catalog service | `internal/server/biz/relay_catalog.go` | Product CRUD and product-channel pool binding | `relay_products`, `relay_product_channels`, `ChannelService` |
| Relay key service | `internal/server/biz/relay_key.go` | Issue, suspend, expire, and bind downstream sub-keys to `api_keys` | `relay_keys`, `APIKeyService` |
| Wallet service | `internal/server/biz/relay_wallet.go` | Recharge, freeze, balance checks, optimistic-lock updates | `relay_wallets`, `relay_wallet_ledger_entries` |
| Access guard | `internal/server/biz/relay_access.go` | Request-time validation of key state, quotas, and balance | `relay_keys`, `relay_wallets`, `relay_daily_usage_summaries` |
| Router service | `internal/server/biz/relay_router.go` | Filter candidate channels by product pool and delegate final selection to `ChannelService` | `relay_product_channels`, `channels`, `provider_quota_status` |
| Settlement service | `internal/server/biz/relay_settlement.go` | Convert `usage_logs` into charges, refunds, and ledger writes | `UsageLogService`, `relay_wallets`, `relay_wallet_ledger_entries` |
| Summary service | `internal/server/biz/relay_usage_summary.go` | Incrementally upsert daily aggregates for dashboards and hard limits | `relay_daily_usage_summaries`, `usage_logs` |
| Admin orchestration | `internal/server/biz/relay_admin.go` | Combine product, key, wallet, and summary data for the operator console | all relay domain services |

### GraphQL and API layering

1. Protocol handlers such as `internal/server/api/openai.go`, `anthropic.go`, and `codex.go` should stay thin and should not write billing logic directly.
2. Relay management should prefer GraphQL, matching the existing console architecture:
   - `internal/server/gql/relay.graphql`
   - `internal/server/gql/relay.resolvers.go`
3. Typical GraphQL capabilities:
   - product list / create / activate / archive
   - product-to-channel pool binding
   - relay key issue / suspend / renew
   - wallet recharge / freeze / manual adjustment
   - ledger, daily summary, and pool usage queries

### Minimal hook points into the current request pipeline

Only 3 integration points should be required:

1. **Post-auth hook**: after API key auth succeeds, load `relay_keys`.
2. **Pre-routing hook**: before final channel selection, filter candidates by product pool.
3. **Post-usage hook**: after `UsageLogService.CreateUsageLog` succeeds, trigger settlement.

That keeps relay business logic out of every protocol handler.

## Query and Index Focus

### Synchronous path queries

1. `api_keys -> relay_keys`: unique lookup by `api_key_id`.
2. `relay_keys -> relay_wallets`: unique lookup by `relay_key_id`.
3. `relay_keys -> relay_daily_usage_summaries`: fetch today's aggregate by `(relay_key_id, stat_date)`.
4. `relay_product_channels -> channels`: fetch candidate pool by `(product_id, status, priority)`.
5. `provider_quota_status`: unique lookup by `channel_id`.

### Asynchronous path queries

1. `usage_logs` grouped by `request_id`, `api_key_id`, and `created_at`.
2. `relay_wallet_ledger_entries` ordered by `relay_key_id + created_at` for billing history.
3. `relay_daily_usage_summaries` scanned by date range for dashboards.

## Recommended Delivery Order

### Phase 1: data structures

1. Add the 6 new Ent schemas.
2. Run `make generate` to produce models and migrations.
3. Ship minimal CRUD and list queries in the operator console first.

### Phase 2: request guard and routing

1. Load `relay_keys` after existing API key auth.
2. Add `RelayAccessService` for state, expiry, balance, and hard-limit validation.
3. Add `RelayRouterService` to filter channel candidates before the existing `ChannelService` selection logic.

### Phase 3: settlement loop

1. Trigger settlement after `usage_logs` are written successfully.
2. Use an idempotency key to prevent duplicate debits.
3. Update the wallet snapshot synchronously and the daily summary asynchronously.

### Phase 4: operator console

1. Product management
2. Pool binding
3. Sub-key management
4. Wallet recharge, freeze, and reconciliation views

## Explicitly Deferred Capabilities

The following should stay out of the MVP:

- third-party payment callback tables and payment gateway integration
- reseller hierarchy and revenue sharing
- self-service signup, coupons, and package versioning
- multi-region capacity pools and cross-region scheduling
- end-user invoice export and tax fields
- request-level preauthorization and partial refund strategies

## Conclusion

The core of this MVP is not a rewrite of AxonHub's request pipeline. It is a relay domain layer on top of the existing `api_keys + channels + requests + usage_logs` model, adding products, sub-keys, wallets, ledgers, and daily summaries.

With this split:

- the synchronous request path only adds a few indexed lookups;
- upstream routing still reuses the existing `ChannelService` and `ProviderQuotaService`;
- accounting and selling rules remain contained inside relay-specific modules instead of polluting the generic request model;
- later expansion to subscriptions, payment callbacks, or reseller flows does not require a new primary data model.
