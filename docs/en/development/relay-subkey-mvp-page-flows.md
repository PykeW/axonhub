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

| Role               | Typical Identity               | Main Pages                                                                                    | Core Actions                                                                                       | Not in MVP                                         |
| ------------------ | ------------------------------ | --------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Platform operator  | Relay owner, admin             | Product pages, channel pool pages, sub-key list, billing pages, request troubleshooting pages | Create products, bind channels, issue/freeze/archive keys, recharge, refund, inspect requests      | Automated pricing experiments, reseller settlement |
| Project admin      | Buyer team owner               | Project product access page, sub-key detail, usage and billing page                           | View available products, copy access credentials, rename keys, inspect balance, limits, and expiry | Bind raw upstream provider credentials             |
| Developer          | Engineer calling the relay API | Getting started page, key detail page, request log snippets                                   | Copy base URL and API key, diagnose recent failures                                                | Change billing policy, manage the shared pool      |
| Support / risk ops | Operator support role          | Key detail, ledger page, request trace page                                                   | Manually freeze, add notes, refund, explain failures                                               | Separate approval workflow                         |

Implementation note: for the MVP, reuse the existing `project_id` isolation and current admin role model first. Add page-level action guards instead of designing a brand-new ACL system.

## Key State Model

### Persisted states

`relay_keys.status` should use the four persisted states defined in the backend design:

| State       | UI meaning        | Request behavior                                    | Allowed actions                             |
| ----------- | ----------------- | --------------------------------------------------- | ------------------------------------------- |
| `active`    | Normal and usable | Request continues after synchronous checks pass     | Recharge, rename, suspend, archive          |
| `suspended` | Manually paused   | Request is rejected immediately                     | Resume, archive, add note                   |
| `exhausted` | Depleted          | Rejected because balance or hard quota is exhausted | Recharge to recover, adjust limits, archive |
| `archived`  | Archived          | Permanently rejected and hidden from default lists  | View history only                           |

### Runtime-derived states

These may not require separate persistence, but they must appear as clear badges in the UI:

| Derived state            | Source                                                            | UI purpose                                                                        |
| ------------------------ | ----------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `expired`                | `expires_at < now()`                                              | Show that the key must be renewed or reissued                                     |
| `low_balance`            | `relay_wallets.available_amount` under threshold                  | Warn before the key becomes unusable                                              |
| `quota_reached`          | `relay_daily_usage_summaries` or monthly aggregates exceed limits | Explain why the key entered `exhausted`                                           |
| `concurrency_blocked`    | Current in-flight usage exceeds `concurrency_limit`               | Explain request failures caused by concurrent demand                              |
| `upstream_pool_degraded` | The product's candidate pool is too small or fully unhealthy      | Show that the problem is in shared upstream capacity, not the buyer's own balance |

Implementation note: the list page should show both persisted status and derived badges. This prevents upstream pool problems from being mistaken for customer balance issues.

## Page Inventory

### Operator-side pages

| Page                                   | Suggested Route                      | Main Role                  | Data Dependencies                                                               | Core Actions                                                                           |
| -------------------------------------- | ------------------------------------ | -------------------------- | ------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Shared-capacity product list           | `/console/relay/products`            | Platform operator          | `relay_products`                                                                | Inspect status, activate/archive, open details                                         |
| Product detail and channel pool config | `/console/relay/products/:id`        | Platform operator          | `relay_products`, `relay_product_channels`, `channels`, `provider_quota_status` | Edit product info, bind/unbind channels, reorder priority and weight, constrain models |
| Sub-key list                           | `/console/relay/keys`                | Platform operator, support | `relay_keys`, `relay_wallets`, project info                                     | Filter by status, search project, spot low-balance and expired keys                    |
| Sub-key detail                         | `/console/relay/keys/:id`            | Platform operator, support | `relay_keys`, `relay_wallets`, `relay_daily_usage_summaries`, recent `requests` | Suspend, resume, archive, rename, adjust expiry                                        |
| Recharge and ledger page               | `/console/relay/keys/:id/billing`    | Platform operator, support | `relay_wallets`, `relay_wallet_ledger_entries`                                  | Recharge, refund, manual adjustment, inspect accounting evidence                       |
| Request trace page                     | `/console/relay/requests`            | Platform operator, support | `requests`, `request_executions`, `usage_logs`                                  | Troubleshoot failures by key, product, or channel, inspect settlement outcome          |
| Channel pool health dashboard          | `/console/relay/channel-pool-health` | Platform operator          | `relay_product_channels`, `channels`, `provider_quota_status`                   | Detect shared upstream capacity risk by product                                        |

### Buyer project pages

| Page                   | Suggested Route                          | Main Role                | Data Dependencies                                                                     | Core Actions                                                          |
| ---------------------- | ---------------------------------------- | ------------------------ | ------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Product access page    | `/projects/:projectId/relay/products`    | Project admin            | Project-visible products, product description, model scope                            | Understand what products are available and navigate to key management |
| Project sub-key list   | `/projects/:projectId/relay/keys`        | Project admin            | `relay_keys`, `relay_wallets`                                                         | View usable keys and copy access info                                 |
| Project sub-key detail | `/projects/:projectId/relay/keys/:id`    | Project admin, developer | `relay_keys`, wallet snapshot, recent requests, SDK examples                          | Copy API key and base URL, inspect status, see recent failures        |
| Usage and billing page | `/projects/:projectId/relay/usage`       | Project admin            | `relay_daily_usage_summaries`, `relay_wallet_ledger_entries`, aggregated `usage_logs` | Inspect balance, consumption trend, and recent charges                |
| Getting started page   | `/projects/:projectId/relay/get-started` | Developer                | Allowed models, request samples, error-code explanations                              | Integrate quickly and understand the shared-capacity model            |

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

| Trigger            | From                       | To                      | Main UI Entry                |
| ------------------ | -------------------------- | ----------------------- | ---------------------------- |
| Create key         | None                       | `active` or `suspended` | Create sub-key dialog        |
| Balance exhausted  | `active`                   | `exhausted`             | Automatic, no manual UI step |
| Recharge recovery  | `exhausted`                | `active`                | Balance and Ledger tab       |
| Manual suspension  | `active` / `exhausted`     | `suspended`             | Key detail page              |
| Resume usage       | `suspended`                | `active`                | Key detail page              |
| Archive            | Any non-archived state     | `archived`              | Key detail page              |
| Renew after expiry | `active` + `expired` badge | `active`                | Key detail page expiry edit  |

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

The current repo is closer to a “directory + `route.tsx` / `index.tsx`” TanStack Router file-based style than to flattening an entire path into a dotted filename. The docs should distinguish these layers clearly:

- `URL route tree / information architecture`: user-facing navigation and page responsibility.
- Actual `frontend/src/routes` implementation: stay close to the existing `__root`, `_authenticated`, and `_authenticated/project` structure.
- If project context continues to come from `ProjectGuard` and the active project selection, project-side pages can live under `_authenticated/project/relay-subkeys/**` first instead of introducing a real `$projectId` segment during the MVP.

A practical implementation-oriented naming example:

```text
frontend/src/routes/
  __root.tsx
  _authenticated/
    relay-subkeys/
      route.tsx
      products/
        index.tsx
        create.tsx
        $productId.tsx
      keys/
        index.tsx
        create.tsx
        $keyId.tsx
        $keyId.billing.tsx
      requests/
        index.tsx
      channel-pool-health/
        index.tsx

    project/
      relay-subkeys/
        route.tsx
        overview.tsx
        products.tsx
        keys/
          index.tsx
          $keyId.tsx
        usage.tsx
        get-started.tsx
        verify.tsx
```

### Page Responsibility and Flow Mapping

#### Operator-side pages

| Page                          | Route                                         | Related Flow                                  | Core Responsibility                                                 |
| ----------------------------- | --------------------------------------------- | --------------------------------------------- | ------------------------------------------------------------------- |
| Product list                  | `/operator/relay-subkeys/products`            | Operator creates shared-capacity products     | Inspect status, filter, activate/archive, open details              |
| Product create                | `/operator/relay-subkeys/products/create`     | Operator creates shared-capacity products     | Enter product code, name, provider type, and model scope            |
| Product detail                | `/operator/relay-subkeys/products/:productId` | Product detail and channel-pool configuration | Manage pool binding, priority, weight, and model filter             |
| Sub-key list                  | `/operator/relay-subkeys/keys`                | Operator issues sub-keys to projects          | View all keys, filter states, identify low-balance and expired keys |
| Sub-key create                | `/operator/relay-subkeys/keys/create`         | Operator issues sub-keys to projects          | Pick project, product, balance mode, expiry, and hard limits        |
| Sub-key detail                | `/operator/relay-subkeys/keys/:keyId`         | Key lifecycle management                      | Inspect overview, recent failures, and execute management actions   |
| Billing page                  | `/operator/relay-subkeys/keys/:keyId/billing` | Recharge and ledger flow                      | Recharge, refund, manually adjust, inspect accounting evidence      |
| Request trace page            | `/operator/relay-subkeys/requests`            | Shared-pool troubleshooting                   | Search failures by key, product, or channel                         |
| Channel pool health dashboard | `/operator/relay-subkeys/channel-pool-health` | Shared-pool troubleshooting                   | Inspect product-pool health and upstream capacity risk              |

#### Project-side pages

| Page                 | Route                                           | Related Flow                              | Core Responsibility                                                 |
| -------------------- | ----------------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------- |
| Overview             | `/projects/:projectId/relay-subkey/overview`    | Project admin inspects and adopts the key | Summarize products, keys, balances, and recent failures             |
| Product access page  | `/projects/:projectId/relay-subkey/products`    | Project admin reviews product scope       | Show project-visible products and model descriptions                |
| Key list             | `/projects/:projectId/relay-subkey/keys`        | Project admin inspects and adopts the key | View usable keys and copy access info                               |
| Key detail           | `/projects/:projectId/relay-subkey/keys/:keyId` | Project admin inspects and adopts the key | Copy API key and base URL, inspect status and recent failures       |
| Usage page           | `/projects/:projectId/relay-subkey/usage`       | Request succeeds and settlement completes | Inspect balance, consumption trend, and recent charges              |
| Getting started page | `/projects/:projectId/relay-subkey/get-started` | Project admin inspects and adopts the key | Provide SDK examples, error-code explanations, and onboarding hints |
| Verify page          | `/projects/:projectId/relay-subkey/verify`      | Integration verification                  | Run lightweight verification and show recent verification results   |

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

### Practical Frontend Follow-up

#### Suggested `frontend/src/routes` File Map

| Page                             | Suggested route file                                                       | Purpose                                                      |
| -------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Operator product list            | `frontend/src/routes/_authenticated/relay-subkeys/products/index.tsx`      | List filters and status-change entry point                   |
| Operator product detail          | `frontend/src/routes/_authenticated/relay-subkeys/products/$productId.tsx` | Product header plus channel-pool / model / assigned-key tabs |
| Operator key list                | `frontend/src/routes/_authenticated/relay-subkeys/keys/index.tsx`          | Central place for status, balance, and expiry filters        |
| Operator key detail              | `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.tsx`         | Overview, limits, recent failures, and management actions    |
| Operator billing page            | `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.billing.tsx` | Recharge, refund, and ledger details                         |
| Operator request troubleshooting | `frontend/src/routes/_authenticated/relay-subkeys/requests/index.tsx`      | Filter requests by product, key, and channel                 |
| Project overview                 | `frontend/src/routes/_authenticated/project/relay-subkeys/overview.tsx`    | Summarize products, keys, balances, and failure highlights   |
| Project key detail               | `frontend/src/routes/_authenticated/project/relay-subkeys/keys/$keyId.tsx` | Access info, recent calls, and failure explanations          |

#### Page-Level Query / Mutation Matrix

| Page                        | Primary queries                                            | Primary mutations                                                                                                         |
| --------------------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Product list / create       | `useRelayProductsQuery`                                    | `useCreateRelayProductMutation`                                                                                           |
| Product detail              | `useRelayProductDetailQuery`, `useRelayChannelPoolQuery`   | `useUpdateRelayProductMutation`, `useBindRelayChannelMutation`                                                            |
| Key list / create           | `useRelayKeysQuery`                                        | `useCreateRelayKeyMutation`                                                                                               |
| Key detail                  | `useRelayKeyDetailQuery`, `useRelayWalletQuery`            | `useSuspendRelayKeyMutation`, `useResumeRelayKeyMutation`, `useArchiveRelayKeyMutation`, `useAdjustRelayKeyLimitMutation` |
| Key billing page            | `useRelayWalletQuery`, `useRelayLedgerEntriesQuery`        | `useRechargeRelayWalletMutation`                                                                                          |
| Request troubleshooting     | `useRelayRequestTraceQuery`                                | `-`                                                                                                                       |
| Channel-pool health         | `useRelayChannelPoolHealthQuery`                           | `-`                                                                                                                       |
| Project overview / products | `useProjectRelayOverviewQuery`, `listProjectRelayProducts` | `-`                                                                                                                       |
| Project key list / detail   | `listProjectRelayKeys`, `getProjectRelayKeyDetail`         | `-`                                                                                                                       |
| Project usage / onboarding  | `getProjectRelayUsageSummary`                              | `-`                                                                                                                       |

#### Implementation Notes Aligned with Current TanStack Router Style

- Let `route.tsx` own section-level layout, `AuthGuard` / `ProjectGuard` wrapping, and shared page chrome; keep leaf route files thin and import the matching feature entry.
- Put list filters, tabs, and pagination in `validateSearch` plus `Route.useSearch()`, matching the current pattern used in `frontend/src/routes/_authenticated/system/index.tsx`.
- Prefer the existing `/project/*` context pattern on project-facing pages first; only promote `:projectId` from documentation semantics into a real path segment after the repo introduces explicit project URL routing.
- After mutations, invalidate only the nearest list/detail queries; show the one-time plaintext key via local page state or a post-create dialog instead of a long-lived store.
- Keep the real page implementations in `frontend/src/features/relay-subkeys/**`; route files should mainly handle guards, search-param parsing, and feature mounting.

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

### Page-Level loader / action / search Param Drafts

#### Operator product list page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/products/index.tsx`
- `validateSearch`
  - `status?: 'draft' | 'active' | 'archived'`
  - `providerType?: string`
  - `keyword?: string`
  - `page?: number`
  - `pageSize?: number`
- loader
  - Parse search params and prefetch the list data behind `useRelayProductsQuery`.
  - Return normalized filters, default pagination config, and permission hints for product creation.
- action / mutation entry points
  - Do not define a TanStack action in the page for MVP; trigger `useCreateRelayProductMutation` or status-switch mutations from buttons.
  - Bulk activation / archival can remain deferred.

#### Operator product detail page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/products/$productId.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'pool' | 'models' | 'keys'`
  - `channelStatus?: 'active' | 'paused' | 'all'`
- loader
  - Prefetch `useRelayProductDetailQuery` and `useRelayChannelPoolQuery` by `productId`.
  - If `tab === 'keys'`, optionally prefetch the assigned-key list as well.
- action / mutation entry points
  - `useUpdateRelayProductMutation`
  - `useBindRelayChannelMutation`
  - Reordering can either use a dedicated mutation or a full-table save in the first release.

#### Operator key list page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/keys/index.tsx`
- `validateSearch`
  - `status?: 'active' | 'suspended' | 'exhausted' | 'archived'`
  - `productId?: string`
  - `projectId?: string`
  - `lowBalanceOnly?: boolean`
  - `expiredOnly?: boolean`
  - `page?: number`
  - `pageSize?: number`
- loader
  - Prefetch `useRelayKeysQuery` and normalize filter values for the page.
  - Optionally prefetch product and project selector options.
- action / mutation entry points
  - Create flow triggers `useCreateRelayKeyMutation`.
  - Bulk suspend / resume should remain out of MVP.

#### Operator key detail page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'limits' | 'failures'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - Prefetch `useRelayKeyDetailQuery` and `useRelayWalletQuery`.
  - When `tab === 'failures'`, optionally prefetch a recent failed-request summary.
- action / mutation entry points
  - `useSuspendRelayKeyMutation`
  - `useResumeRelayKeyMutation`
  - `useArchiveRelayKeyMutation`
  - `useAdjustRelayKeyLimitMutation`
- note
  - The one-time plaintext key should not come back from a normal loader. It should be handled by local page state or a post-create dialog only.

#### Operator billing page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.billing.tsx`
- `validateSearch`
  - `scene?: 'all' | 'recharge' | 'consume' | 'refund' | 'manual_adjust'`
  - `page?: number`
  - `pageSize?: number`
- loader
  - Prefetch `useRelayWalletQuery` and `useRelayLedgerEntriesQuery`.
- action / mutation entry points
  - `useRechargeRelayWalletMutation`
  - If refunds and manual adjustments are open in MVP, expose them as separate mutations.

#### Operator request troubleshooting page

- Target route file: `frontend/src/routes/_authenticated/relay-subkeys/requests/index.tsx`
- `validateSearch`
  - `productId?: string`
  - `keyId?: string`
  - `channelId?: string`
  - `status?: 'failed' | 'completed' | 'all'`
  - `timeRange?: '1h' | '24h' | '7d'`
  - `page?: number`
- loader
  - Prefetch `useRelayRequestTraceQuery`.
  - Prefetch product, key, and channel selector options from the same search context.
- action / mutation entry points
  - No MVP page-level action; retry, suspend, or archive should redirect to the owning key or product screens.

#### Project overview page

- Target route file: `frontend/src/routes/_authenticated/project/relay-subkeys/overview.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'products' | 'keys'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - After `ProjectGuard` passes, prefetch `useProjectRelayOverviewQuery`.
  - If the current project comes from route context, reuse that context instead of duplicating a project identifier in search params.
- action / mutation entry points
  - None; this page stays read-only in the first release.

#### Project key detail page

- Target route file: `frontend/src/routes/_authenticated/project/relay-subkeys/keys/$keyId.tsx`
- `validateSearch`
  - `tab?: 'access' | 'usage' | 'failures'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - Prefetch `getProjectRelayKeyDetail`.
  - If `tab === 'usage'`, optionally prefetch recent ledger or token-aggregate data.
- action / mutation entry points
  - Do not expose write mutations on the project side in MVP; copying the key, copying the base URL, and running verification remain local UI actions.

#### Project getting-started page

- Target route file: `frontend/src/routes/_authenticated/project/relay-subkeys/get-started.tsx`
- `validateSearch`
  - `provider?: 'openai' | 'anthropic' | 'codex'`
  - `keyId?: string`
- loader
  - Prefetch visible products, the recommended default key, and SDK-example metadata.
- action / mutation entry points
  - None; if a `verify integration` button exists, it should navigate to the verify page or trigger a lightweight verification mutation.

#### Unified implementation guidance

- Standardize all search params through `validateSearch`, then drive filters, tabs, and pagination via `Route.useSearch()`.
- Keep authorization in `route.tsx`, `AuthGuard`, and `ProjectGuard`; loaders should not duplicate access checks beyond assuming guarded context.
- Let loaders focus on first-paint data and search normalization; ongoing invalidation and refetching still belong to TanStack Query.
- If TanStack Router actions are introduced later, reserve them for simple form-submit flows; for the MVP, mutation hooks remain the closer fit to the current codebase.

### UI States and Empty-State Guidance

#### List pages

- `loading`: render skeleton rows while keeping the filter bar visible so the page does not look blank.
- `empty`: provide an explicit CTA such as `Create Product` or `Contact operator to issue a key`.
- `filtered-empty`: preserve current filters and offer a one-click reset.
- `error`: keep the last search params intact and show a retry affordance instead of resetting the page state.

#### Detail pages

- `loading`: render the header card skeleton first and lazy-load tab content.
- `not-found`: if the product or key no longer exists, redirect back to the list page with a toast.
- `forbidden`: show an explicit permission error rather than masking it as a 404.
- `partial-degraded`: if the main entity loads but a secondary query fails, scope the error UI to that block instead of breaking the entire page.

#### Action feedback

- `success`: use toast plus lightweight local refresh after create, suspend, resume, recharge, or similar mutations.
- `submitting`: set the active button to loading and prevent duplicate submission.
- `failure`: surface backend-provided, explainable errors first, such as insufficient balance, quota conflict, or degraded shared pool state.

### Query Invalidation and Refresh Guidance

#### Product domain

- After product creation: invalidate `useRelayProductsQuery`.
- After product update: invalidate both `useRelayProductDetailQuery` and `useRelayProductsQuery`.
- After channel-pool changes: invalidate `useRelayChannelPoolQuery`, and invalidate product detail as needed.

#### Key domain

- After key creation: invalidate `useRelayKeysQuery`; if creation happens inside product detail, also invalidate the assigned-key list there.
- After suspend / resume / archive: invalidate `useRelayKeyDetailQuery` and `useRelayKeysQuery`.
- After limit adjustment: invalidate only the current key detail query unless the list page also exposes limit summaries.

#### Wallet and ledger domain

- After recharge / refund / manual adjustment: invalidate `useRelayWalletQuery`, `useRelayLedgerEntriesQuery`, and, when needed, the related key detail query.
- If the project overview shows aggregated balance, also invalidate `useProjectRelayOverviewQuery`.

#### Project-side read-only domain

- Project overview and usage pages should load on entry and avoid aggressive polling by default.
- Only enable short-interval refresh for focused tabs such as `recent requests` or `recent failures` while the user is actively viewing them.

### Draft Acceptance Criteria

#### Operator product list page

- Supports filtering by status, provider type, and keyword.
- URL search params remain the source of truth, and refresh preserves the same list state.
- A newly created product becomes visible in the list without manual hard refresh.

#### Operator product detail page

- Supports switching among `overview`, `pool`, `models`, and `keys` tabs.
- Channel-pool edits refresh both detail content and health summary consistently.
- If the full upstream pool is unavailable, the page clearly explains why the product cannot move to `active`.

#### Operator key list and detail pages

- Distinguishes all four persisted states: `active`, `suspended`, `exhausted`, and `archived`.
- Exposes derived badges such as `expired`, `low_balance`, `quota_reached`, and `upstream_pool_degraded`.
- Suspend, resume, and archive operations produce consistent list/detail state after completion.

#### Operator billing page

- Balance summary, ledger table, and recharge action refresh coherently.
- After recharge, updated balance and the latest ledger row appear without manual reload.
- Failures show backend explanations rather than failing silently.

#### Project overview / key detail / getting-started pages

- Project-facing pages do not expose operator-only actions.
- Project users can reliably see Base URL, masked key, recommended models, and recent failure hints.
- Switching provider examples on the getting-started page updates both search params and code examples consistently.

### Documentation and Implementation Sync Notes

- If frontend routing later moves to an explicit `projects/:projectId/*` structure, update the current notes about reusing project context.
- If the app later adopts TanStack Router loader/action patterns more broadly, revise this section so the current “mutation-hook driven” assumption matches reality.
- If approval or request workflows are introduced, the current resource-centric route tree should be refactored into explicit request, approval, and binding flows, and the acceptance criteria should be rewritten accordingly.

### UI Copy Library

#### Empty-state copy

| Scenario                        | Title                                     | Supporting copy                                                                         | Primary action       |
| ------------------------------- | ----------------------------------------- | --------------------------------------------------------------------------------------- | -------------------- |
| Product list is empty           | No shared-capacity products yet           | Create a product first, then bind upstream channels and issue sellable keys.            | Create Product       |
| Key list is empty               | No sub-keys available yet                 | Buyers can only connect after at least one key has been created.                        | Create Sub-Key       |
| Project has no visible products | No products are available to this project | Contact the platform operator to assign a product or verify project scope.              | Contact Operator     |
| Filtered list is empty          | No matching results                       | No records match the current filters. Try clearing filters and searching again.         | Clear Filters        |
| Recent requests is empty        | No request history yet                    | Recent requests and failure summaries will appear here after the first successful call. | View Getting Started |

#### Toast copy

| Scenario          | Success copy                           | Failure copy                                                   |
| ----------------- | -------------------------------------- | -------------------------------------------------------------- |
| Create product    | Product created successfully           | Failed to create product. Please try again.                    |
| Update product    | Product settings saved                 | Failed to save product settings. Check input and retry.        |
| Bind channel pool | Channel added to the shared pool       | Failed to bind channel. Verify channel status and permissions. |
| Create key        | Sub-key created successfully           | Failed to create sub-key. Check project and product settings.  |
| Suspend key       | Key suspended                          | Failed to suspend key. Please try again.                       |
| Resume key        | Key resumed                            | Failed to resume key. Check current status and retry.          |
| Archive key       | Key archived                           | Failed to archive key. Please try again.                       |
| Recharge          | Recharge completed and balance updated | Recharge failed. Check amount or retry later.                  |
| Copy Base URL     | Base URL copied                        | Copy failed. Please copy manually.                             |
| Copy key          | Key copied                             | Copy failed. Please copy manually.                             |

#### Error-message copy

| Error type              | Suggested copy                                                                                     |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| Insufficient balance    | This key does not have enough balance. Please contact the operator to recharge it.                 |
| Daily quota reached     | This key has reached its daily limit. Try again tomorrow or switch to another key.                 |
| Key suspended           | This key is currently suspended. Please contact the platform operator for details.                 |
| Key archived            | This key has been archived and can no longer accept new requests.                                  |
| Shared pool unavailable | The shared pool is temporarily unavailable. Please retry later or ask the operator to investigate. |
| Forbidden               | You do not have permission to view this page or perform this action.                               |
| Not found               | The requested product or key could not be found. It may have been deleted or archived.             |
| Load failed             | Failed to load data. Please try again later.                                                       |

#### Dangerous-action confirmation copy

| Action         | Title                | Confirmation copy                                                                                        | Confirm button |
| -------------- | -------------------- | -------------------------------------------------------------------------------------------------------- | -------------- |
| Suspend key    | Suspend this key?    | Once suspended, all new requests will be rejected immediately, while historical data remains available.  | Suspend Key    |
| Resume key     | Resume this key?     | Once resumed, the key will participate in validation and routing again.                                  | Resume Key     |
| Archive key    | Archive this key?    | Once archived, the key will disappear from default lists and no longer accept new requests.              | Archive Key    |
| Remove channel | Remove this channel? | Removing the channel may reduce available capacity. Confirm that other channels are still healthy first. | Remove Channel |

#### Skeleton / loading copy guidance

- Product list: keep the filter bar visible and only skeletonize the table area.
- Product detail: render the header card skeleton first, then lazy-load tab sections.
- Key detail: show the status summary card first; load billing and failure blocks independently.
- Project getting-started page: the code example area may show “Generating example configuration...” while loading.

#### Empty / error interaction rules

- Empty states should always suggest the next available action instead of stopping at “no data”.
- Error states should preserve search params and user input whenever possible.
- Never reuse the same copy for `forbidden` and `not-found`.
- For shared-pool incidents, explain that the issue may be in the pool state, not necessarily in the current key itself.

### Button Copy and Disabled-State Matrix

#### Primary action buttons

| Button         | Default copy   | Loading copy  | Disabled when                                                                          |
| -------------- | -------------- | ------------- | -------------------------------------------------------------------------------------- |
| Create Product | Create Product | Creating...   | Form validation fails, required fields are missing, or the user lacks write permission |
| Save Settings  | Save Settings  | Saving...     | No changes exist, validation fails, or channel-pool config is invalid                  |
| Create Sub-Key | Create Sub-Key | Creating...   | No project selected, no product bound, or key limit inputs are invalid                 |
| Save Limits    | Save Limits    | Saving...     | Limits are unchanged or values are outside the allowed range                           |
| Recharge Now   | Recharge Now   | Recharging... | Amount is empty, below the minimum amount, or the user lacks billing permission        |
| Reload         | Reload         | Reloading...  | The same request is already in flight                                                  |

#### Dangerous action buttons

| Button         | Default copy   | Loading copy  | Disabled when                                                                 |
| -------------- | -------------- | ------------- | ----------------------------------------------------------------------------- |
| Suspend Key    | Suspend Key    | Suspending... | Key is already `suspended` or `archived`                                      |
| Resume Key     | Resume Key     | Resuming...   | Key is not currently `suspended` or `exhausted`                               |
| Archive Key    | Archive Key    | Archiving...  | Key is already `archived`                                                     |
| Remove Channel | Remove Channel | Removing...   | The product only has one healthy channel left and no fallback capacity exists |

#### Project-side read-only buttons

| Button               | Default copy         | Loading copy | Disabled when                                                    |
| -------------------- | -------------------- | ------------ | ---------------------------------------------------------------- |
| Copy Base URL        | Copy Base URL        | Copying...   | Base URL is empty                                                |
| Copy Key             | Copy Key             | Copying...   | The current page is not allowed to reveal or copy the key        |
| View Getting Started | View Getting Started | Opening...   | Never                                                            |
| Verify Integration   | Verify Integration   | Verifying... | No usable key exists or the shared pool is currently unavailable |

#### Disabled-state interaction rules

- Keep disabled buttons visible rather than hiding them, so users understand the action exists but is currently unavailable.
- For permission-based disablement, prefer a tooltip such as `You do not have permission to perform this action`.
- For state-based disablement, the tooltip should explain the concrete reason whenever possible, such as `This key is already archived` or `No removable channel is available`.
- For incomplete form state, do not stop at a vague disabled state; explain the missing prerequisite, such as `Select a project and product first`.

#### Secondary button copy suggestions

- Back to list: `Back to List`
- View details: `View Details`
- Clear filters: `Clear Filters`
- Expand all: `Expand All`
- Collapse: `Collapse`
- View recent failures: `View Recent Failures`
- View ledger: `View Ledger`

### Table Column Titles and Tooltip Copy Library

#### Product list table

| Column title        | Tooltip copy                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------------------- |
| Product Name        | The display name used to identify a shared-capacity product across operator and project views.                      |
| Provider Type       | Identifies the provider or protocol family served by the product, such as Codex, Claude Code, or OpenAI-compatible. |
| Status              | Shows whether the product is currently in draft, active, or archived state.                                         |
| Allowed Model Count | The number of models currently enabled for this product.                                                            |
| Pool Health         | A summary of the bound channel pool's current availability, used for quick shared-pool triage.                      |
| Last Updated        | The latest time when product configuration was changed.                                                             |

#### Key list table

| Column title      | Tooltip copy                                                                              |
| ----------------- | ----------------------------------------------------------------------------------------- |
| Key Name          | The display name of the sub-key, used to distinguish keys by project or purpose.          |
| Project           | The project context that owns the current key.                                            |
| Bound Product     | The shared-capacity product currently attached to the key.                                |
| Status            | The persisted lifecycle state, such as `active`, `suspended`, `exhausted`, or `archived`. |
| Available Balance | The balance currently available for settlement, excluding any frozen amount.              |
| Expiry Time       | The time when the key becomes invalid and should trigger renewal or warning behavior.     |
| Last Used         | The most recent time when the key entered the request path successfully.                  |
| Derived State     | Runtime badges such as `expired`, `low_balance`, or `upstream_pool_degraded`.             |

#### Ledger table

| Column title    | Tooltip copy                                                                              |
| --------------- | ----------------------------------------------------------------------------------------- |
| Ledger Time     | The timestamp when this ledger entry was recorded.                                        |
| Type            | Indicates whether the entry is a recharge, charge, refund, or manual adjustment.          |
| Amount          | The amount changed by this entry; interpret direction together with the entry type.       |
| Balance After   | The balance snapshot after this entry was applied.                                        |
| Related Request | If this entry came from request settlement, this field links back to the related request. |
| Operator        | The operator account that triggered a manual accounting action.                           |
| Remark          | A human-written or system-generated note explaining the entry.                            |

#### Request troubleshooting table

| Column title    | Tooltip copy                                                                                      |
| --------------- | ------------------------------------------------------------------------------------------------- |
| Request Time    | The time when the request entered the platform request pipeline.                                  |
| Product         | The shared-capacity product associated with the request.                                          |
| Key             | The sub-key used by the request.                                                                  |
| Target Model    | The model identifier requested by the user.                                                       |
| Routed Channel  | The upstream channel that actually executed the request.                                          |
| Response Status | The final outcome of the request, such as success, failed, or rejected.                           |
| Failure Stage   | If failed, indicates whether the failure happened during auth, routing, execution, or settlement. |
| Charge Result   | Whether charging completed successfully for this request.                                         |
| Trace ID        | The unique identifier used to connect tracing, troubleshooting, and log lookup.                   |

#### Project-side usage table

| Column title       | Tooltip copy                                                                               |
| ------------------ | ------------------------------------------------------------------------------------------ |
| Date               | The aggregation date for the current usage record.                                         |
| Request Count      | Total request count in the current aggregation window.                                     |
| Token Usage        | Total settled token count, used to observe consumption trends.                             |
| Balance Change     | Net balance change during the selected time window.                                        |
| Top Failure Reason | The most common failure summary in the current period, helping buyer-side troubleshooting. |

#### Tooltip usage rules

- Tooltips should explain field meaning first, not repeat the column title.
- If a value is already fully self-explanatory, tooltip text can be omitted to reduce hover noise.
- For state columns, prioritize explaining both the meaning and user impact.
- For accounting columns, prioritize explaining the amount definition and whether the value already reflects frozen or post-settlement state.

### SDK Integration Feedback Copy

#### Copy feedback messages

| Scenario                    | Success copy                      | Failure copy                                                 |
| --------------------------- | --------------------------------- | ------------------------------------------------------------ |
| Copy Base URL               | Base URL copied                   | Failed to copy Base URL. Please copy it manually.            |
| Copy Sub-Key                | Sub-Key copied                    | Failed to copy the Sub-Key. Please copy it manually.         |
| Copy current example        | Example copied                    | Failed to copy the current example. Please copy it manually. |
| Switch to OpenAI example    | Switched to the OpenAI example    | Failed to load the OpenAI example. Please try again.         |
| Switch to Anthropic example | Switched to the Anthropic example | Failed to load the Anthropic example. Please try again.      |
| Switch to Codex example     | Switched to the Codex example     | Failed to load the Codex example. Please try again.          |

#### Integration validation and test-request messages

| Scenario                                    | Copy                                                                                              |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Validation started                          | Validating the current integration settings. Please wait...                                       |
| Validation succeeded                        | Validation succeeded. This Sub-Key can access the shared-capacity service normally.               |
| Validation failed (Sub-Key auth)            | Validation failed: check whether the Sub-Key is correct, active, and not expired.                 |
| Validation failed (missing config)          | Validation failed: Base URL, Sub-Key, or model is missing. Fill in the required fields and retry. |
| Validation failed (insufficient permission) | Validation failed: you do not have permission to verify this key or view its full configuration.  |
| Validation failed (model unavailable)       | Validation failed: the selected model is not enabled for this product. Try another model.         |
| Validation failed (network)                 | Validation failed: the request did not complete. Check your network and retry later.              |
| Test request in progress                    | Sending a test request through the shared pool. This may take a few seconds...                    |
| Test request succeeded                      | Test request succeeded. The relay accepted the Sub-Key and returned a valid response.             |
| Test request failed (pool unavailable)      | Test request failed: the shared pool is temporarily unavailable. Please retry later.              |
| Test request degraded (fallback route used) | Test request completed through a fallback route because the primary relay pool is degraded.       |

#### Integration guidance copy

- Always use the AxonHub downstream Sub-Key, not the upstream provider's native credential.
- If validation or the test request fails after copying the values, first verify that Base URL, Sub-Key, and model name match the example shown on the page.
- When the relay pool is degraded, show whether the request failed because the pool is unavailable or succeeded through fallback capacity.
- If validation keeps failing, contact the platform operator and provide the latest request time and Trace ID.

#### Example-switching copy

- OpenAI Example
- Anthropic Example
- Codex Example
- Copy Current Example
- Switched to the {provider} example
- Failed to switch examples. Please try again.

#### SDK page empty/error-state suggestions

- No usable Sub-Key: no Sub-Key is currently available for this project. Contact the operator to issue one.
- Missing configuration: the current key, Base URL, or model configuration is incomplete. Refresh the page or contact the operator.
- No available model: no model is currently available for this product. Refresh later or confirm with the operator.
- Example generation failed: failed to load example configuration. Please try again later.
- Insufficient permission: you do not have permission to view this key's full SDK configuration.
- Relay pool degraded: the shared relay pool is degraded. Requests may fail over to backup capacity or return temporary errors.
- No verification history: the latest validation result will appear here after the first successful verification.
