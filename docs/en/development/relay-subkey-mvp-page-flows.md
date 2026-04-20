# Self-Hosted Relay + Sub-Key Shared-Capacity MVP: Page Inventory and User Flows

## Background

AxonHub already has the API gateway, channel routing, request auditing, and API key authentication needed for a self-hosted relay model. For the "self-hosted relay + sub-key shared-capacity" MVP, the goal is not to redesign the protocol layer. The goal is to add the minimum operator-facing and buyer-facing console surfaces needed to sell, operate, and troubleshoot shared upstream capacity.

Paired with the backend design document, the page layer should revolve around these objects:

- `relay_products`: sellable shared-capacity products.
- `relay_product_channels`: upstream channel pools bound to a product.
- `relay_keys`: downstream sub-keys issued to customers.
- `relay_wallets` / `relay_wallet_ledger_entries`: balance snapshots and accounting events.
- `relay_daily_usage_summaries`: lightweight quota and dashboard aggregates.
- `requests` / `request_executions` / `usage_logs`: request facts, routed channel facts, and settlement facts.

## MVP Goals

1. Let operators complete the full loop of creating products, binding upstream channels, issuing keys, recharging balances, freezing keys, and troubleshooting requests.
2. Let buyer-side project admins view product scope, receive usable sub-keys, and track balance, hard limits, and expiry.
3. Let developers keep using existing OpenAI / Anthropic / Codex-compatible access patterns without a new protocol branch.
4. Keep the UI focused on MVP actions only; defer online payments, reseller workflows, multi-seller marketplace flows, and complex support systems.

## Roles and Permission Boundaries

| Role | Typical Identity | Main Pages | Core Actions | Not in MVP |
| --- | --- | --- | --- | --- |
| Platform operator | Relay owner, admin | Product pages, channel pool pages, sub-key list, billing pages, request troubleshooting pages | Create products, bind channels, issue/freeze/archive keys, recharge, refund, inspect requests | Automated pricing experiments, reseller settlement |
| Project admin | Buyer team owner | Project product access page, sub-key detail, usage and billing page | View available products, copy access credentials, rename keys, inspect balance, limits, and expiry | Bind raw upstream provider credentials |
| Developer | Engineer calling the relay API | Getting started page, key detail page, request log snippets | Copy base URL and API key, diagnose recent failures | Change billing policy, manage the shared pool |
| Support / risk ops | Operator support role | Key detail, ledger page, request trace page | Manually freeze, add notes, refund, explain failures | Separate approval workflow |

Implementation note: for the MVP, reuse the existing `project_id` isolation and current admin role model first. Add page-level action guards instead of designing a brand-new ACL system.

## Key State Model

### Persisted states

`relay_keys.status` should use the four persisted states defined in the backend design:

| State | UI meaning | Request behavior | Allowed actions |
| --- | --- | --- | --- |
| `active` | Normal and usable | Request continues after synchronous checks pass | Recharge, rename, suspend, archive |
| `suspended` | Manually paused | Request is rejected immediately | Resume, archive, add note |
| `exhausted` | Depleted | Rejected because balance or hard quota is exhausted | Recharge to recover, adjust limits, archive |
| `archived` | Archived | Permanently rejected and hidden from default lists | View history only |

### Runtime-derived states

These may not require separate persistence, but they must appear as clear badges in the UI:

| Derived state | Source | UI purpose |
| --- | --- | --- |
| `expired` | `expires_at < now()` | Show that the key must be renewed or reissued |
| `low_balance` | `relay_wallets.available_amount` under threshold | Warn before the key becomes unusable |
| `quota_reached` | `relay_daily_usage_summaries` or monthly aggregates exceed limits | Explain why the key entered `exhausted` |
| `concurrency_blocked` | Current in-flight usage exceeds `concurrency_limit` | Explain request failures caused by concurrent demand |
| `upstream_pool_degraded` | The product's candidate pool is too small or fully unhealthy | Show that the problem is in shared upstream capacity, not the buyer's own balance |

Implementation note: the list page should show both persisted status and derived badges. This prevents upstream pool problems from being mistaken for customer balance issues.

## Page Inventory

### Operator-side pages

| Page | Suggested Route | Main Role | Data Dependencies | Core Actions |
| --- | --- | --- | --- | --- |
| Shared-capacity product list | `/console/relay/products` | Platform operator | `relay_products` | Inspect status, activate/archive, open details |
| Product detail and channel pool config | `/console/relay/products/:id` | Platform operator | `relay_products`, `relay_product_channels`, `channels`, `provider_quota_status` | Edit product info, bind/unbind channels, reorder priority and weight, constrain models |
| Sub-key list | `/console/relay/keys` | Platform operator, support | `relay_keys`, `relay_wallets`, project info | Filter by status, search project, spot low-balance and expired keys |
| Sub-key detail | `/console/relay/keys/:id` | Platform operator, support | `relay_keys`, `relay_wallets`, `relay_daily_usage_summaries`, recent `requests` | Suspend, resume, archive, rename, adjust expiry |
| Recharge and ledger page | `/console/relay/keys/:id/billing` | Platform operator, support | `relay_wallets`, `relay_wallet_ledger_entries` | Recharge, refund, manual adjustment, inspect accounting evidence |
| Request trace page | `/console/relay/requests` | Platform operator, support | `requests`, `request_executions`, `usage_logs` | Troubleshoot failures by key, product, or channel, inspect settlement outcome |
| Channel pool health dashboard | `/console/relay/channel-pool-health` | Platform operator | `relay_product_channels`, `channels`, `provider_quota_status` | Detect shared upstream capacity risk by product |

### Buyer project pages

| Page | Suggested Route | Main Role | Data Dependencies | Core Actions |
| --- | --- | --- | --- | --- |
| Product access page | `/projects/:projectId/relay/products` | Project admin | Project-visible products, product description, model scope | Understand what products are available and navigate to key management |
| Project sub-key list | `/projects/:projectId/relay/keys` | Project admin | `relay_keys`, `relay_wallets` | View usable keys and copy access info |
| Project sub-key detail | `/projects/:projectId/relay/keys/:id` | Project admin, developer | `relay_keys`, wallet snapshot, recent requests, SDK examples | Copy API key and base URL, inspect status, see recent failures |
| Usage and billing page | `/projects/:projectId/relay/usage` | Project admin | `relay_daily_usage_summaries`, `relay_wallet_ledger_entries`, aggregated `usage_logs` | Inspect balance, consumption trend, and recent charges |
| Getting started page | `/projects/:projectId/relay/get-started` | Developer | Allowed models, request samples, error-code explanations | Integrate quickly and understand the shared-capacity model |

### Page composition guidance

To keep routing small in the MVP, prefer a list page plus a multi-tab detail page:

- Product detail tabs: `Overview`, `Channel Pool`, `Allowed Models`, `Assigned Keys`.
- Key detail tabs: `Overview`, `Balance and Ledger`, `Requests`, `Limits`.
- Reuse as many operator components as possible on project pages and hide privileged actions.

## Critical Fields Per Page

### 1. Product detail page

Must show:

- Product code, name, provider type, and status.
- Allowed models, default timeout, and billing mode.
- Bound channel count, healthy channel count, and current unavailability reason.
- Per-channel priority, weight, model filter, and latest quota status.

### 2. Key list and key detail pages

Must show:

- Key display name, owning project, bound product, and current status.
- Masked API key, creation time, expiry time, and last-used time.
- Available balance, frozen amount, daily request count, daily tokens, and monthly cost.
- Recent failure summary such as insufficient balance, daily quota reached, or upstream pool unavailable.

### 3. Request troubleshooting page

Must show:

- Request time, project, key, product, and requested model.
- Routed `channel_id`, upstream provider, latency, and response status code.
- Whether charging happened, charge amount, and linked `usage_log_id` / ledger entry.
- Failure stage: auth failure, key validation failure, routing failure, upstream execution failure, or settlement failure.

## End-to-End User Flows

### Flow 1: Operator creates a shared-capacity product

1. The operator opens the shared-capacity product list and clicks `Create Product`.
2. They enter product code, name, provider type, billing mode, allowed models, and default timeout.
3. After saving, they open the product detail page and bind one or more `channels` in the `Channel Pool` tab.
4. For each binding, they set `priority`, `weight`, `model_filter`, and `allow_fallback`.
5. The page shows live pool health; if every channel is unavailable, the product cannot be switched to `active`.
6. Once activated, the product becomes visible to project admins or can be used immediately when the operator issues keys.

Implementation note: save basic product metadata through `relay_products`, and keep channel-pool edits as separate `relay_product_channels` operations so the frontend can support reorder and partial updates cleanly.

### Flow 2: Operator issues a sub-key to a project

1. The operator opens the sub-key list and clicks `Create Sub-Key`.
2. They choose the target `project_id`, bound product, display name, expiry, balance mode, and hard limits.
3. The backend creates the `api_keys` record and then creates `relay_keys` and `relay_wallets`.
4. The UI lands on the key detail page and shows the plaintext key once for copy.
5. If the product uses prepaid balance, the operator can immediately perform an initial recharge in the `Balance and Ledger` tab.

Implementation note: the post-create screen must explicitly warn that the plaintext key is only shown once. All later views should display a masked value.

### Flow 3: Project admin inspects and integrates the key

1. The project admin finds the issued key in the project sub-key list.
2. On the detail page, they inspect the base URL, supported models, SDK samples, and current status hints.
3. They configure the key in an existing OpenAI / Anthropic / Codex client.
4. After the first successful request, the page updates `last_used_at` and shows a success entry in the recent requests area.

Implementation note: the getting-started content must clearly explain that this is an AxonHub downstream key, not a raw upstream provider credential.

### Flow 4: Request succeeds and settlement completes

1. The client calls a compatible API endpoint using the AxonHub sub-key.
2. Auth middleware validates `api_keys`, then loads `relay_keys` and `relay_wallets`.
3. If the key is valid and its balance and quotas pass checks, routing loads the product's channel pool using `product_id`.
4. The routing layer filters unavailable channels, selects the final `channel_id`, and writes `requests` and `request_executions`.
5. After the upstream call succeeds, settlement uses `usage_logs` to create ledger entries and update the wallet snapshot.
6. Both project-side and operator-side pages can now show the request, including model, tokens, charge amount, and routed channel.

Implementation note: the UI should separate `request succeeded` from `charge settled` so a delayed settlement job does not look like a missing record.

### Flow 5: Low balance or hard quota exhausts the key

1. A request arrives and the system detects `available_amount <= 0`, or a daily/monthly limit has already been reached.
2. The backend marks the key as `exhausted` and returns an explainable error.
3. The key list shows `exhausted` plus a `low_balance` or `quota_reached` badge.
4. The project admin sees the failure reason and the last trigger time in the key detail page.
5. After the operator recharges the wallet or adjusts limits, the key returns to `active`.

Implementation note: `exhausted` must remain a recoverable state and should not share wording with `suspended`. The UI actions should also distinguish `recharge to recover` from `manual unfreeze`.

### Flow 6: Operator manually suspends or archives a key

1. An operator or support user clicks `Suspend Key` on the key detail page.
2. The system updates `relay_keys.status` to `suspended` and requires an operator note.
3. Later requests are rejected immediately, and the request page reports the failure stage as `key validation failure`.
4. If the key is being retired permanently, the operator archives it so it disappears from default lists while keeping history available.

Implementation note: the suspend action should require a note so project-side pages can show a specific explanation instead of a generic 403-style message.

### Flow 7: Operator troubleshoots a shared pool outage

1. Multiple projects report errors on the same product.
2. The operator opens the channel pool health dashboard to see whether all bound channels for that product are degraded.
3. They move to the request trace page and filter failures by product and time window.
4. By comparing `request_executions` and `provider_quota_status`, they determine whether the issue is quota exhaustion, channel disablement, or upstream API instability.
5. Immediate mitigations include pausing the bad channel, changing healthy channel priority, restricting a model, or temporarily moving the product back to `draft` or a paused state.

Implementation note: the UI needs both `problem by key` and `problem by product / shared pool` entry points. Otherwise, shared-capacity failures will be misdiagnosed as single-user incidents.

## Suggested State Transitions

| Trigger | From | To | Main UI Entry |
| --- | --- | --- | --- |
| Create key | None | `active` or `suspended` | Create sub-key dialog |
| Balance exhausted | `active` | `exhausted` | Automatic, no manual UI step |
| Recharge recovery | `exhausted` | `active` | Balance and Ledger tab |
| Manual suspension | `active` / `exhausted` | `suspended` | Key detail page |
| Resume usage | `suspended` | `active` | Key detail page |
| Archive | Any non-archived state | `archived` | Key detail page |
| Renew after expiry | `active` + `expired` badge | `active` | Key detail page expiry edit |

## Out of Scope for the MVP UI

These pages should stay out of the first version:

- Online payment checkout, order center, and invoice download.
- Self-service purchasing and shopping-cart flows.
- Upstream credential marketplace or seller center.
- Complex approval systems, support tickets, or reconciliation export centers.
- Multi-level reseller hierarchy and revenue-sharing settlement pages.

## Recommended Delivery Order

### Phase 1: Smallest shippable loop

1. Product list plus product detail with channel pool config.
2. Sub-key list plus key detail.
3. Balance and ledger tab.
4. Project-side key list plus getting-started page.

### Phase 2: Operational hardening

1. Request trace page.
2. Channel pool health dashboard.
3. Richer status badges and clearer failure reason exposure.

This order gets the team to a usable "sell shared capacity and let customers actually call it" loop first, then improves troubleshooting depth and operator efficiency.

## Frontend Route Tree and Page Decomposition

### Design Goals

- Split the information architecture into operator-side and project-side tracks, reducing single-page state-machine complexity.
- Keep URL semantics stable so the MVP can grow into richer operational, governance, and troubleshooting flows without route churn.
- Stay compatible with AxonHub's existing TanStack Router and file-based routing style, without forcing URL structure and final file names to match exactly.
- In the first release, prioritize the loop of product creation -> channel-pool binding -> sub-key issuance -> project onboarding -> request troubleshooting; defer secondary capabilities into tabs or embedded sections.

### Recommended Route Tree

The visible URL information architecture should be organized around two main tracks:

```text
/operator/relay-subkeys
  /products
  /products/create
  /products/:productId
  /products/:productId/edit
  /keys
  /keys/create
  /keys/:keyId
  /keys/:keyId/billing
  /keys/:keyId/requests
  /channel-pool-health
  /requests

/projects/:projectId/relay-subkey
  /overview
  /products
  /keys
  /keys/:keyId
  /usage
  /get-started
  /verify
```

Notes:
- The `operator` track serves platform operators, support, and risk-control roles.
- The `projects/:projectId/relay-subkey` track serves buyer-side project admins and developers.
- If the MVP needs additional scope control, only `products`, `keys`, `overview`, and `get-started` need to become standalone routes first; the remaining capabilities can live in tabs.

### File-Based Route Naming Guidance

The documentation should explicitly distinguish two layers:
- `URL route tree / information architecture`: user-facing navigation and page responsibility.
- `file-based route naming`: the implementation that hangs under the existing `__root`, `_auth`, operator layout, and project layout structure.

A reasonable implementation-oriented naming example:

```text
frontend/src/routes/
  __root.tsx
  _auth.operator.relay-subkeys.products.index.tsx
  _auth.operator.relay-subkeys.products.create.tsx
  _auth.operator.relay-subkeys.products.$productId.index.tsx
  _auth.operator.relay-subkeys.keys.index.tsx
  _auth.operator.relay-subkeys.keys.create.tsx
  _auth.operator.relay-subkeys.keys.$keyId.index.tsx
  _auth.operator.relay-subkeys.keys.$keyId.billing.tsx
  _auth.operator.relay-subkeys.requests.tsx
  _auth.operator.relay-subkeys.channel-pool-health.tsx

  _auth.projects.$projectId.relay-subkey.overview.tsx
  _auth.projects.$projectId.relay-subkey.products.tsx
  _auth.projects.$projectId.relay-subkey.keys.tsx
  _auth.projects.$projectId.relay-subkey.keys.$keyId.tsx
  _auth.projects.$projectId.relay-subkey.usage.tsx
  _auth.projects.$projectId.relay-subkey.get-started.tsx
  _auth.projects.$projectId.relay-subkey.verify.tsx
```

### Page Responsibility and Flow Mapping

#### Operator-side pages

| Page | Route | Related Flow | Core Responsibility |
| --- | --- | --- | --- |
| Product list | `/operator/relay-subkeys/products` | Operator creates shared-capacity products | Inspect status, filter, activate/archive, open details |
| Product create | `/operator/relay-subkeys/products/create` | Operator creates shared-capacity products | Enter product code, name, provider type, and model scope |
| Product detail | `/operator/relay-subkeys/products/:productId` | Product detail and channel-pool configuration | Manage pool binding, priority, weight, and model filter |
| Sub-key list | `/operator/relay-subkeys/keys` | Operator issues sub-keys to projects | View all keys, filter states, identify low-balance and expired keys |
| Sub-key create | `/operator/relay-subkeys/keys/create` | Operator issues sub-keys to projects | Pick project, product, balance mode, expiry, and hard limits |
| Sub-key detail | `/operator/relay-subkeys/keys/:keyId` | Key lifecycle management | Inspect overview, recent failures, and execute management actions |
| Billing page | `/operator/relay-subkeys/keys/:keyId/billing` | Recharge and ledger flow | Recharge, refund, manually adjust, inspect accounting evidence |
| Request trace page | `/operator/relay-subkeys/requests` | Shared-pool troubleshooting | Search failures by key, product, or channel |
| Channel pool health dashboard | `/operator/relay-subkeys/channel-pool-health` | Shared-pool troubleshooting | Inspect product-pool health and upstream capacity risk |

#### Project-side pages

| Page | Route | Related Flow | Core Responsibility |
| --- | --- | --- | --- |
| Overview | `/projects/:projectId/relay-subkey/overview` | Project admin inspects and adopts the key | Summarize products, keys, balances, and recent failures |
| Product access page | `/projects/:projectId/relay-subkey/products` | Project admin reviews product scope | Show project-visible products and model descriptions |
| Key list | `/projects/:projectId/relay-subkey/keys` | Project admin inspects and adopts the key | View usable keys and copy access info |
| Key detail | `/projects/:projectId/relay-subkey/keys/:keyId` | Project admin inspects and adopts the key | Copy API key and base URL, inspect status and recent failures |
| Usage page | `/projects/:projectId/relay-subkey/usage` | Request succeeds and settlement completes | Inspect balance, consumption trend, and recent charges |
| Getting started page | `/projects/:projectId/relay-subkey/get-started` | Project admin inspects and adopts the key | Provide SDK examples, error-code explanations, and onboarding hints |
| Verify page | `/projects/:projectId/relay-subkey/verify` | Integration verification | Run lightweight verification and show recent verification results |

### Page and Component Decomposition

#### Page-level components

Operator-side:
- `RelayProductListPage`
- `RelayProductCreatePage`
- `RelayProductDetailPage`
- `RelayKeyListPage`
- `RelayKeyCreatePage`
- `RelayKeyDetailPage`
- `RelayKeyBillingPage`
- `RelayRequestTracePage`
- `RelayChannelPoolHealthPage`

Project-side:
- `ProjectRelayOverviewPage`
- `ProjectRelayProductAccessPage`
- `ProjectRelayKeyListPage`
- `ProjectRelayKeyDetailPage`
- `ProjectRelayUsagePage`
- `ProjectRelayGetStartedPage`
- `ProjectRelayVerifyPage`

#### Shared business components

Product-related:
- `RelayProductTable`
- `RelayProductStatusBadge`
- `RelayProductForm`
- `RelayChannelPoolTable`
- `RelayChannelPoolEditor`
- `RelayChannelHealthSummary`
- `AllowedModelList`

Key-related:
- `RelayKeyTable`
- `RelayKeyStatusBadge`
- `RelayKeyCreateForm`
- `RelayKeyOverviewCard`
- `RelayKeyMaskedSecretCard`
- `RelayKeyLimitPanel`
- `RelayKeyDerivedStateBadges`

Billing and usage:
- `RelayWalletSummaryCard`
- `RelayLedgerTable`
- `RelayRechargeDialog`
- `RelayUsageSummaryChart`
- `RelayUsageFilters`
- `RelayChargeResultBadge`

Request troubleshooting:
- `RelayRequestTable`
- `RelayRequestFilters`
- `RelayExecutionTimeline`
- `RelayFailureStageBadge`
- `RelayRequestDetailDrawer`

Project onboarding:
- `RelaySdkExampleCard`
- `RelayBaseUrlCard`
- `RelayVerifyPanel`
- `RelayErrorGuide`

#### Recommended decomposition inside large pages

- `RelayProductDetailPage`
  - `RelayProductHeader`
  - `RelayProductOverviewTab`
  - `RelayProductChannelPoolTab`
  - `RelayProductAllowedModelsTab`
  - `RelayProductAssignedKeysTab`
- `RelayKeyDetailPage`
  - `RelayKeyHeader`
  - `RelayKeyOverviewTab`
  - `RelayKeyBillingTab`
  - `RelayKeyRequestTab`
  - `RelayKeyLimitTab`
- `ProjectRelayKeyDetailPage`
  - `ProjectRelayKeyHeader`
  - `ProjectRelayKeyAccessTab`
  - `ProjectRelayKeyUsageTab`
  - `ProjectRelayKeyRecentFailuresTab`

### Data Loading and State Management Guidance

Recommended usage:
- Page-level remote data: TanStack Query
- Lightweight cross-page UI state: Zustand
- Forms: React Hook Form plus Zod
- Table filters: URL search params

Suggested query hooks:
- `useRelayProductsQuery`
- `useRelayProductDetailQuery`
- `useRelayChannelPoolQuery`
- `useRelayKeysQuery`
- `useRelayKeyDetailQuery`
- `useRelayWalletQuery`
- `useRelayLedgerEntriesQuery`
- `useRelayRequestTraceQuery`
- `useRelayChannelPoolHealthQuery`
- `useProjectRelayOverviewQuery`
- `useProjectRelayUsageQuery`

Suggested mutation hooks:
- `useCreateRelayProductMutation`
- `useUpdateRelayProductMutation`
- `useBindRelayChannelMutation`
- `useCreateRelayKeyMutation`
- `useSuspendRelayKeyMutation`
- `useResumeRelayKeyMutation`
- `useArchiveRelayKeyMutation`
- `useRechargeRelayWalletMutation`
- `useAdjustRelayKeyLimitMutation`

Suggested Zustand stores:
- `useRelayProductFilterStore`
- `useRelayKeyFilterStore`
- `useRelayRequestTraceFilterStore`
- `useRelayUiStore`

Guidance: business truth should stay in query results; stores should hold UI state and lightweight filter state only.

### GraphQL / API Boundary Suggestions

The relay management console should continue to prefer a GraphQL-first boundary. Suggested domain grouping:

Product domain:
- `listRelayProducts`
- `getRelayProduct`
- `createRelayProduct`
- `updateRelayProduct`
- `archiveRelayProduct`

Channel-pool domain:
- `listRelayProductChannels`
- `bindRelayProductChannel`
- `unbindRelayProductChannel`
- `reorderRelayProductChannels`

Sub-key domain:
- `listRelayKeys`
- `getRelayKey`
- `createRelayKey`
- `suspendRelayKey`
- `resumeRelayKey`
- `archiveRelayKey`
- `renewRelayKey`

Wallet domain:
- `getRelayWallet`
- `listRelayLedgerEntries`
- `rechargeRelayWallet`
- `refundRelayWallet`
- `adjustRelayWallet`

Troubleshooting domain:
- `listRelayRequests`
- `getRelayRequestDetail`
- `listRelayChannelPoolHealth`

Project-side domain:
- `getProjectRelayOverview`
- `listProjectRelayProducts`
- `listProjectRelayKeys`
- `getProjectRelayKeyDetail`
- `getProjectRelayUsageSummary`

### MVP Route Boundaries

In the first release, only the following pages need to become standalone routes; the rest can remain tabs or drawers:

Operator-side:
- `/operator/relay-subkeys/products`
- `/operator/relay-subkeys/products/:productId`
- `/operator/relay-subkeys/keys`
- `/operator/relay-subkeys/keys/:keyId`

Project-side:
- `/projects/:projectId/relay-subkey/overview`
- `/projects/:projectId/relay-subkey/keys/:keyId`
- `/projects/:projectId/relay-subkey/get-started`

Benefits:
- Small route count.
- Simpler permission checks.
- Faster MVP delivery.
- Future extraction of `billing`, `requests`, and `channel-pool-health` into standalone pages without rewriting URL semantics.

### Post-MVP Expansion Directions

Potential future standalone routes:
- Richer request troubleshooting views.
- A dedicated channel-pool health dashboard.
- Product approval and publishing flows.
- Bulk key management pages.
- Project-level verification center.
- Custom notification and alert pages.
