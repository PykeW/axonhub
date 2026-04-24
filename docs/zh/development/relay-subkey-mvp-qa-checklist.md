# Relay/Sub-Key MVP QA Checklist

Use this checklist to keep the self-hosted Relay + Sub-Key shared-capacity MVP aligned while backend data, runtime hooks, and docs land in parallel.

## Contract Fixture

The backend contract test pins the operator-facing Relay product constants to `internal/server/biz/testdata/relay_product_contract.json`:

- Provider types: `claudecode`, `codex`, `openai_compatible`
- Access mode: `shared_capacity`
- Billing modes: `prepaid`, `quota_only`
- Product statuses: `draft`, `active`, `archived`
- Channel binding statuses: `active`, `paused`
- Defaults: `USD`, `shared_capacity`, `prepaid`, `draft`, timeout `600`, binding weight `100`

## Data Foundation Checks

- `internal/ent/schema/relay_product.go` defines products without storing upstream credentials.
- `internal/ent/schema/relay_product_channel.go` maps products to shared upstream channel pools.
- `internal/server/biz/relay_product.go` keeps validation available before generated Ent artifacts are refreshed.
- CRUD methods may return `ErrRelayProductCodegenRequired` until Ent generation is completed.

## Targeted Verification

Run these after touching the data foundation or QA fixture:

```bash
gofmt -w internal/server/biz/relay_product.go internal/server/biz/relay_product_test.go internal/server/biz/relay_product_contract_test.go
GOTOOLCHAIN=local go test ./internal/server/biz -run RelayProduct -count=1
```

If local Go is older than the `go.mod` requirement, report the exact toolchain blocker instead of downloading a toolchain in this worktree.

## Runtime Coordination

- Avoid editing runtime hook files from the data/QA slice unless the runtime teammate coordinates it.
- Keep product/channel-pool constants synchronized with the backend design and page-flow docs.
- Treat contract fixture drift as a review blocker for frontend option lists and API docs.
