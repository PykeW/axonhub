import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from '@/lib/api-client';
import { useSelectedProjectId } from '@/stores/projectStore';

export type RelayProductStatus = 'draft' | 'active' | 'archived';
export type RelayBillingMode = 'prepaid' | 'quota_only';
export type RelayProviderType = 'claudecode' | 'codex' | 'openai_compatible';
export type RelayChannelBindingStatus = 'active' | 'paused';
export type RelayHealthStatus = 'healthy' | 'degraded' | 'unavailable';
export type RelayKeyStatus = 'active' | 'suspended' | 'exhausted' | 'archived';
export type RelayDerivedState = 'expired' | 'low_balance' | 'quota_reached' | 'concurrency_blocked' | 'upstream_pool_degraded';
export type RelayWalletLedgerType = 'recharge' | 'charge' | 'refund' | 'adjustment' | 'freeze' | 'unfreeze';
export type RelayFailureStage = 'auth' | 'key_validation' | 'routing' | 'upstream' | 'settlement' | 'none';
export type RelaySettlementStatus = 'not_started' | 'charged' | 'delayed' | 'failed' | 'skipped';

export interface RelayProductChannel {
  id: string;
  productId?: string;
  channelId: string;
  channelName: string;
  provider: RelayProviderType;
  priority: number;
  weight: number;
  status: RelayChannelBindingStatus;
  health: RelayHealthStatus;
  allowFallback: boolean;
  modelFilter: string[];
  quotaRemainingPercent: number;
  errorRatePercent: number;
  latencyMs: number;
  unavailableReason?: string;
  lastCheckedAt: string;
}

export interface RelayProduct {
  id: string;
  code: string;
  name: string;
  providerType: RelayProviderType;
  status: RelayProductStatus;
  billingMode: RelayBillingMode;
  description: string;
  allowedModels: string[];
  defaultTimeoutMs: number;
  poolHealth: RelayHealthStatus;
  channelPool: RelayProductChannel[];
  keyCount: number;
  activeKeyCount: number;
  monthlyRequestCount: number;
  monthlyTokenCount: number;
  monthlyCost: number;
  createdAt: string;
  updatedAt: string;
}

export interface RelayKeyLimitSnapshot {
  dailyRequestLimit: number;
  dailyTokenLimit: number;
  monthlyCostLimit: number;
  concurrencyLimit: number;
}

export interface RelayKeyUsageSnapshot {
  todayRequests: number;
  todayTokens: number;
  monthlyCost: number;
  lastFailureAt?: string;
  recentFailure?: string;
}

export interface RelayKey {
  id: string;
  apiKeyId: string;
  plaintextKey?: string;
  projectId: string;
  projectName: string;
  productId: string;
  productName: string;
  name: string;
  maskedKey: string;
  status: RelayKeyStatus;
  derivedStates: RelayDerivedState[];
  balanceMode: RelayBillingMode;
  expiresAt?: string;
  createdAt: string;
  lastUsedAt?: string;
  baseUrl: string;
  limits: RelayKeyLimitSnapshot;
  usage: RelayKeyUsageSnapshot;
}

export interface RelayWallet {
  id: string;
  relayKeyId: string;
  currency: string;
  availableAmount: number;
  frozenAmount: number;
  totalRecharged: number;
  totalSpent: number;
  creditLimit: number;
  lowBalanceThreshold: number;
  updatedAt: string;
}

export interface RelayWalletLedgerEntry {
  id: string;
  relayKeyId: string;
  type: RelayWalletLedgerType;
  amount: number;
  currency: string;
  balanceAfter: number;
  referenceId?: string;
  operator: string;
  note: string;
  createdAt: string;
}

export interface RelayDailyUsageSummary {
  relayKeyId: string;
  statDate: string;
  requests: number;
  promptTokens: number;
  completionTokens: number;
  totalCost: number;
}

export interface RelayRequestTrace {
  id: string;
  requestId: string;
  createdAt: string;
  projectName: string;
  keyName: string;
  productName: string;
  modelId: string;
  channelName?: string;
  provider?: RelayProviderType;
  status: 'completed' | 'failed' | 'processing';
  failureStage: RelayFailureStage;
  latencyMs?: number;
  responseStatusCode?: number;
  charged: boolean;
  chargeAmount: number;
  settlementStatus: RelaySettlementStatus;
  usageLogId?: string;
  ledgerEntryId?: string;
  promptTokens: number;
  completionTokens: number;
  errorMessage?: string;
}

export interface RelayChannelPoolHealth {
  productId: string;
  productName: string;
  status: RelayHealthStatus;
  healthyChannels: number;
  degradedChannels: number;
  unavailableChannels: number;
  riskReason?: string;
  channels: RelayProductChannel[];
}

export interface ProjectRelayOverview {
  products: RelayProduct[];
  keys: RelayKey[];
  wallets: RelayWallet[];
  recentRequests: RelayRequestTrace[];
  usage: RelayDailyUsageSummary[];
}

export interface ProjectRelayUsage {
  wallets: RelayWallet[];
  ledgerEntries: RelayWalletLedgerEntry[];
  usage: RelayDailyUsageSummary[];
  recentRequests: RelayRequestTrace[];
}

export interface CreateRelayProductInput {
  code: string;
  name: string;
  providerType: RelayProviderType;
  billingMode: RelayBillingMode;
  allowedModels: string[];
  defaultTimeoutMs: number;
  description?: string;
}

export interface UpdateRelayProductInput extends Partial<CreateRelayProductInput> {
  status?: RelayProductStatus;
}

export interface BindRelayChannelInput {
  productId: string;
  channelId: string;
  priority: number;
  weight: number;
  modelFilter: string[];
  allowFallback: boolean;
}

export interface UpdateRelayChannelBindingInput {
  id: string;
  productId?: string;
  priority?: number;
  weight?: number;
  status?: RelayChannelBindingStatus;
  modelFilter?: string[];
  allowFallback?: boolean;
}

export interface CreateRelayKeyInput {
  projectId: string;
  productId: string;
  name: string;
  expiresAt?: string;
  balanceMode: RelayBillingMode;
  initialBalance: number;
  limits: RelayKeyLimitSnapshot;
}

export interface RechargeRelayWalletInput {
  relayKeyId: string;
  amount: number;
  note: string;
}
// Legacy Relay/Sub-Key admin API surface for the older operator-managed flow.
// Keep compatibility fixes only; new Share/Use work should not add fresh dependencies here.
export type RelaySubkeysApiMode = 'mock' | 'rest';

const RELAY_SUBKEYS_API_BASE = '/admin/relay-subkeys';

const RELAY_SUBKEYS_PROJECT_API_BASE = '/admin/projects';

const relayApiMode = (): RelaySubkeysApiMode => (import.meta.env.VITE_RELAY_SUBKEYS_API_MODE === 'rest' ? 'rest' : 'mock');
const relayApiEnabled = () => relayApiMode() === 'rest';
export const relaySubkeysApiModeLabel = () =>
  relayApiEnabled()
    ? 'REST API'
    : 'Mock fallback (set VITE_RELAY_SUBKEYS_API_MODE=rest only after backend REST contract is available)';

const iso = (day: string, time: string) => `2026-04-${day}T${time}:00Z`;

type RelayApiRequestOptions = NonNullable<Parameters<typeof apiRequest>[1]>;

function encodedPath(value: string) {
  return encodeURIComponent(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function unwrapArrayResponse<T>(response: unknown, keys: string[]): T[] | undefined {
  if (Array.isArray(response)) return response as T[];
  if (!isRecord(response)) return undefined;

  for (const key of [...keys, 'data']) {
    const candidate = response[key];
    if (Array.isArray(candidate)) return candidate as T[];
  }

  return undefined;
}

function unwrapEntityResponse<T>(response: unknown, keys: string[]): T | undefined {
  if (!isRecord(response)) return response as T;

  for (const key of [...keys, 'data']) {
    const candidate = response[key];
    if (candidate !== undefined && candidate !== null) return candidate as T;
  }

  return Object.keys(response).length > 0 ? (response as T) : undefined;
}

function relayContractError(resource: string, expected: string) {
  return new Error(`Relay REST contract error: ${resource} response missing ${expected}`);
}

function requireArrayResponse<T>(response: unknown, keys: string[], resource: string): T[] {
  const value = unwrapArrayResponse<T>(response, keys);
  if (value) return value;
  throw relayContractError(resource, `array field (${[...keys, 'data'].join(' | ')})`);
}

function requireEntityResponse<T>(response: unknown, keys: string[], resource: string): T {
  const value = unwrapEntityResponse<T>(response, keys);
  if (value !== undefined && value !== null) return value;
  throw relayContractError(resource, `entity field (${[...keys, 'data'].join(' | ')})`);
}

async function relayApiRequest<T>(endpoint: string, options: RelayApiRequestOptions = {}): Promise<T> {
  return apiRequest<T>(`${RELAY_SUBKEYS_API_BASE}${endpoint}`, { ...options, requireAuth: true });
}

async function relayProjectApiRequest<T>(projectId: string, endpoint: string, options: RelayApiRequestOptions = {}): Promise<T> {
  return apiRequest<T>(`${RELAY_SUBKEYS_PROJECT_API_BASE}/${encodedPath(projectId)}/relay-subkeys${endpoint}`, {
    ...options,
    requireAuth: true,
  });
}

export const mockRelayProducts: RelayProduct[] = [
  {
    id: 'prod-gpt-shared',
    code: 'gpt-shared-pro',
    name: 'GPT Shared Pro',
    providerType: 'openai_compatible',
    status: 'active',
    billingMode: 'prepaid',
    description: 'OpenAI-compatible shared capacity with pooled fallback channels.',
    allowedModels: ['gpt-4.1-mini', 'gpt-4o-mini', 'text-embedding-3-small'],
    defaultTimeoutMs: 60000,
    poolHealth: 'healthy',
    keyCount: 18,
    activeKeyCount: 15,
    monthlyRequestCount: 48210,
    monthlyTokenCount: 38640000,
    monthlyCost: 1542.38,
    createdAt: iso('02', '09:20'),
    updatedAt: iso('28', '06:10'),
    channelPool: [
      {
        id: 'bind-openai-primary',
        channelId: 'ch-openai-primary',
        channelName: 'OpenAI primary US',
        provider: 'openai_compatible',
        priority: 1,
        weight: 70,
        status: 'active',
        health: 'healthy',
        allowFallback: true,
        modelFilter: ['gpt-4.1-mini', 'gpt-4o-mini'],
        quotaRemainingPercent: 72,
        errorRatePercent: 0.8,
        latencyMs: 840,
        lastCheckedAt: iso('28', '06:12'),
      },
      {
        id: 'bind-openai-backup',
        channelId: 'ch-openai-backup',
        channelName: 'OpenAI backup EU',
        provider: 'openai_compatible',
        priority: 2,
        weight: 30,
        status: 'active',
        health: 'healthy',
        allowFallback: true,
        modelFilter: ['gpt-4o-mini', 'text-embedding-3-small'],
        quotaRemainingPercent: 64,
        errorRatePercent: 1.1,
        latencyMs: 1120,
        lastCheckedAt: iso('28', '06:12'),
      },
    ],
  },
  {
    id: 'prod-claude-shared',
    code: 'claude-team-relay',
    name: 'Claude Team Relay',
    providerType: 'claudecode',
    status: 'active',
    billingMode: 'prepaid',
    description: 'Claude Code-compatible shared pool for support and analysis workloads.',
    allowedModels: ['claude-3-5-haiku-latest', 'claude-3-5-sonnet-latest'],
    defaultTimeoutMs: 90000,
    poolHealth: 'degraded',
    keyCount: 9,
    activeKeyCount: 7,
    monthlyRequestCount: 18140,
    monthlyTokenCount: 22480000,
    monthlyCost: 1188.94,
    createdAt: iso('05', '11:35'),
    updatedAt: iso('28', '05:48'),
    channelPool: [
      {
        id: 'bind-anthropic-primary',
        channelId: 'ch-anthropic-primary',
        channelName: 'Anthropic pooled primary',
        provider: 'claudecode',
        priority: 1,
        weight: 80,
        status: 'active',
        health: 'degraded',
        allowFallback: true,
        modelFilter: ['claude-3-5-haiku-latest', 'claude-3-5-sonnet-latest'],
        quotaRemainingPercent: 18,
        errorRatePercent: 5.6,
        latencyMs: 1820,
        unavailableReason: 'Provider quota is close to daily limit.',
        lastCheckedAt: iso('28', '06:05'),
      },
      {
        id: 'bind-anthropic-maintenance',
        channelId: 'ch-anthropic-maintenance',
        channelName: 'Anthropic standby APAC',
        provider: 'claudecode',
        priority: 2,
        weight: 20,
        status: 'paused',
        health: 'unavailable',
        allowFallback: false,
        modelFilter: ['claude-3-5-haiku-latest'],
        quotaRemainingPercent: 0,
        errorRatePercent: 13.2,
        latencyMs: 0,
        unavailableReason: 'Temporarily disabled during provider key rotation.',
        lastCheckedAt: iso('28', '06:05'),
      },
    ],
  },
  {
    id: 'prod-codex-trial',
    code: 'codex-trial-lab',
    name: 'Codex Trial Lab',
    providerType: 'codex',
    status: 'draft',
    billingMode: 'quota_only',
    description: 'Draft product for Codex-compatible integration testing.',
    allowedModels: ['gpt-4.1-mini', 'claude-3-5-haiku-latest'],
    defaultTimeoutMs: 45000,
    poolHealth: 'unavailable',
    keyCount: 2,
    activeKeyCount: 0,
    monthlyRequestCount: 320,
    monthlyTokenCount: 140000,
    monthlyCost: 18.27,
    createdAt: iso('16', '14:45'),
    updatedAt: iso('27', '18:22'),
    channelPool: [],
  },
];

export const mockRelayKeys: RelayKey[] = [
  {
    id: 'rkey-alpha-prod',
    apiKeyId: 'api-alpha-prod',
    projectId: 'project-alpha',
    projectName: 'Alpha Apps',
    productId: 'prod-gpt-shared',
    productName: 'GPT Shared Pro',
    name: 'Production checkout relay',
    maskedKey: 'ahub_sk_live_9Lx...Q2m',
    status: 'active',
    derivedStates: [],
    balanceMode: 'prepaid',
    expiresAt: iso('30', '23:59'),
    createdAt: iso('07', '10:10'),
    lastUsedAt: iso('28', '06:08'),
    baseUrl: 'https://axonhub.example.com/v1',
    limits: { dailyRequestLimit: 50000, dailyTokenLimit: 25000000, monthlyCostLimit: 1800, concurrencyLimit: 24 },
    usage: { todayRequests: 8120, todayTokens: 6420000, monthlyCost: 882.18 },
  },
  {
    id: 'rkey-beta-support',
    apiKeyId: 'api-beta-support',
    projectId: 'project-beta',
    projectName: 'Beta Support',
    productId: 'prod-claude-shared',
    productName: 'Claude Team Relay',
    name: 'Support assistant relay',
    maskedKey: 'ahub_sk_live_R3p...8va',
    status: 'active',
    derivedStates: ['low_balance', 'upstream_pool_degraded'],
    balanceMode: 'prepaid',
    expiresAt: '2026-05-12T23:59:00Z',
    createdAt: iso('10', '12:30'),
    lastUsedAt: iso('28', '05:58'),
    baseUrl: 'https://axonhub.example.com/anthropic',
    limits: { dailyRequestLimit: 22000, dailyTokenLimit: 12000000, monthlyCostLimit: 1200, concurrencyLimit: 12 },
    usage: {
      todayRequests: 6034,
      todayTokens: 7180000,
      monthlyCost: 1094.52,
      lastFailureAt: iso('28', '05:54'),
      recentFailure: 'Shared upstream pool is degraded; fallback channel is draining.',
    },
  },
  {
    id: 'rkey-alpha-expired',
    apiKeyId: 'api-alpha-expired',
    projectId: 'project-alpha',
    projectName: 'Alpha Apps',
    productId: 'prod-gpt-shared',
    productName: 'GPT Shared Pro',
    name: 'Old staging relay',
    maskedKey: 'ahub_sk_test_E1s...0kp',
    status: 'exhausted',
    derivedStates: ['expired', 'quota_reached'],
    balanceMode: 'prepaid',
    expiresAt: '2026-04-20T23:59:00Z',
    createdAt: '2026-03-12T08:20:00Z',
    lastUsedAt: iso('27', '21:30'),
    baseUrl: 'https://axonhub.example.com/v1',
    limits: { dailyRequestLimit: 2000, dailyTokenLimit: 1000000, monthlyCostLimit: 80, concurrencyLimit: 4 },
    usage: {
      todayRequests: 2000,
      todayTokens: 1012000,
      monthlyCost: 80.04,
      lastFailureAt: iso('27', '21:31'),
      recentFailure: 'Daily request limit reached after the key expired.',
    },
  },
  {
    id: 'rkey-gamma-paused',
    apiKeyId: 'api-gamma-paused',
    projectId: 'project-gamma',
    projectName: 'Gamma Lab',
    productId: 'prod-codex-trial',
    productName: 'Codex Trial Lab',
    name: 'Codex lab preview',
    maskedKey: 'ahub_sk_test_M2n...7ra',
    status: 'suspended',
    derivedStates: ['upstream_pool_degraded'],
    balanceMode: 'quota_only',
    createdAt: iso('18', '09:05'),
    baseUrl: 'https://axonhub.example.com/v1',
    limits: { dailyRequestLimit: 1000, dailyTokenLimit: 500000, monthlyCostLimit: 0, concurrencyLimit: 2 },
    usage: {
      todayRequests: 420,
      todayTokens: 131000,
      monthlyCost: 0,
      lastFailureAt: iso('26', '12:45'),
      recentFailure: 'Suspended by operator while trial channel pool is empty.',
    },
  },
];

export const mockRelayWallets: RelayWallet[] = [
  {
    id: 'wallet-alpha-prod',
    relayKeyId: 'rkey-alpha-prod',
    currency: 'USD',
    availableAmount: 917.82,
    frozenAmount: 24.12,
    totalRecharged: 1800,
    totalSpent: 882.18,
    creditLimit: 0,
    lowBalanceThreshold: 150,
    updatedAt: iso('28', '06:08'),
  },
  {
    id: 'wallet-beta-support',
    relayKeyId: 'rkey-beta-support',
    currency: 'USD',
    availableAmount: 105.48,
    frozenAmount: 0,
    totalRecharged: 1200,
    totalSpent: 1094.52,
    creditLimit: 0,
    lowBalanceThreshold: 180,
    updatedAt: iso('28', '05:58'),
  },
  {
    id: 'wallet-alpha-expired',
    relayKeyId: 'rkey-alpha-expired',
    currency: 'USD',
    availableAmount: 0,
    frozenAmount: 0,
    totalRecharged: 80,
    totalSpent: 80.04,
    creditLimit: 0,
    lowBalanceThreshold: 20,
    updatedAt: iso('27', '21:31'),
  },
];

export const mockRelayLedgerEntries: RelayWalletLedgerEntry[] = [
  {
    id: 'ledger-alpha-recharge-1',
    relayKeyId: 'rkey-alpha-prod',
    type: 'recharge',
    amount: 800,
    currency: 'USD',
    balanceAfter: 948.74,
    referenceId: 'manual-topup-1048',
    operator: 'ops@axonhub.local',
    note: 'Initial production top-up after launch approval.',
    createdAt: iso('20', '10:12'),
  },
  {
    id: 'ledger-alpha-charge-1',
    relayKeyId: 'rkey-alpha-prod',
    type: 'charge',
    amount: -2.46,
    currency: 'USD',
    balanceAfter: 917.82,
    referenceId: 'usage-log-8831',
    operator: 'system',
    note: 'Settled request req-relay-8831.',
    createdAt: iso('28', '06:08'),
  },
  {
    id: 'ledger-beta-charge-1',
    relayKeyId: 'rkey-beta-support',
    type: 'charge',
    amount: -4.18,
    currency: 'USD',
    balanceAfter: 105.48,
    referenceId: 'usage-log-7720',
    operator: 'system',
    note: 'Settled support assistant request.',
    createdAt: iso('28', '05:58'),
  },
  {
    id: 'ledger-beta-adjustment-1',
    relayKeyId: 'rkey-beta-support',
    type: 'adjustment',
    amount: 150,
    currency: 'USD',
    balanceAfter: 184.44,
    referenceId: 'support-credit-62',
    operator: 'support@axonhub.local',
    note: 'Manual service credit after provider outage.',
    createdAt: iso('24', '09:30'),
  },
  {
    id: 'ledger-alpha-expired-charge',
    relayKeyId: 'rkey-alpha-expired',
    type: 'charge',
    amount: -0.14,
    currency: 'USD',
    balanceAfter: 0,
    referenceId: 'usage-log-6402',
    operator: 'system',
    note: 'Final settlement before quota exhaustion.',
    createdAt: iso('27', '21:31'),
  },
];

export const mockRelayUsageSummaries: RelayDailyUsageSummary[] = [
  { relayKeyId: 'rkey-alpha-prod', statDate: '2026-04-28', requests: 8120, promptTokens: 3420000, completionTokens: 3000000, totalCost: 92.84 },
  { relayKeyId: 'rkey-alpha-prod', statDate: '2026-04-27', requests: 10320, promptTokens: 4260000, completionTokens: 3840000, totalCost: 118.72 },
  { relayKeyId: 'rkey-beta-support', statDate: '2026-04-28', requests: 6034, promptTokens: 3860000, completionTokens: 3320000, totalCost: 84.28 },
  { relayKeyId: 'rkey-alpha-expired', statDate: '2026-04-27', requests: 2000, promptTokens: 612000, completionTokens: 400000, totalCost: 12.04 },
];

export const mockRelayRequestTraces: RelayRequestTrace[] = [
  {
    id: 'trace-8831',
    requestId: 'req-relay-8831',
    createdAt: iso('28', '06:08'),
    projectName: 'Alpha Apps',
    keyName: 'Production checkout relay',
    productName: 'GPT Shared Pro',
    modelId: 'gpt-4.1-mini',
    channelName: 'OpenAI primary US',
    provider: 'openai_compatible',
    status: 'completed',
    failureStage: 'none',
    latencyMs: 1240,
    responseStatusCode: 200,
    charged: true,
    chargeAmount: 2.46,
    settlementStatus: 'charged',
    usageLogId: 'usage-log-8831',
    ledgerEntryId: 'ledger-alpha-charge-1',
    promptTokens: 16200,
    completionTokens: 11800,
  },
  {
    id: 'trace-7720',
    requestId: 'req-relay-7720',
    createdAt: iso('28', '05:58'),
    projectName: 'Beta Support',
    keyName: 'Support assistant relay',
    productName: 'Claude Team Relay',
    modelId: 'claude-3-5-sonnet-latest',
    channelName: 'Anthropic pooled primary',
    provider: 'claudecode',
    status: 'completed',
    failureStage: 'none',
    latencyMs: 2120,
    responseStatusCode: 200,
    charged: true,
    chargeAmount: 4.18,
    settlementStatus: 'charged',
    usageLogId: 'usage-log-7720',
    ledgerEntryId: 'ledger-beta-charge-1',
    promptTokens: 18400,
    completionTokens: 9300,
  },
  {
    id: 'trace-7719',
    requestId: 'req-relay-7719',
    createdAt: iso('28', '05:54'),
    projectName: 'Beta Support',
    keyName: 'Support assistant relay',
    productName: 'Claude Team Relay',
    modelId: 'claude-3-5-sonnet-latest',
    status: 'failed',
    failureStage: 'routing',
    responseStatusCode: 503,
    charged: false,
    chargeAmount: 0,
    settlementStatus: 'skipped',
    promptTokens: 0,
    completionTokens: 0,
    errorMessage: 'No healthy upstream channel remained after pool filtering.',
  },
  {
    id: 'trace-6402',
    requestId: 'req-relay-6402',
    createdAt: iso('27', '21:31'),
    projectName: 'Alpha Apps',
    keyName: 'Old staging relay',
    productName: 'GPT Shared Pro',
    modelId: 'gpt-4o-mini',
    channelName: 'OpenAI backup EU',
    provider: 'openai_compatible',
    status: 'failed',
    failureStage: 'key_validation',
    responseStatusCode: 402,
    charged: true,
    chargeAmount: 0.14,
    settlementStatus: 'charged',
    usageLogId: 'usage-log-6402',
    ledgerEntryId: 'ledger-alpha-expired-charge',
    promptTokens: 900,
    completionTokens: 640,
    errorMessage: 'Daily request limit reached and key is now exhausted.',
  },
];

export const relayQueryKeys = {
  products: ['relay-subkeys', 'products'] as const,
  product: (productId: string | undefined) => ['relay-subkeys', 'products', productId] as const,
  channelPool: (productId: string | undefined) => ['relay-subkeys', 'channel-pool', productId] as const,
  keys: ['relay-subkeys', 'keys'] as const,
  key: (keyId: string | undefined) => ['relay-subkeys', 'keys', keyId] as const,
  wallet: (keyId: string | undefined) => ['relay-subkeys', 'wallet', keyId] as const,
  ledger: (keyId: string | undefined) => ['relay-subkeys', 'ledger', keyId] as const,
  requests: ['relay-subkeys', 'requests'] as const,
  poolHealth: ['relay-subkeys', 'pool-health'] as const,
  projectOverview: (projectId: string | null | undefined) => ['relay-subkeys', 'project', projectId, 'overview'] as const,
  projectProducts: (projectId: string | null | undefined) => ['relay-subkeys', 'project', projectId, 'products'] as const,
  projectKeys: (projectId: string | null | undefined) => ['relay-subkeys', 'project', projectId, 'keys'] as const,
  projectUsage: (projectId: string | null | undefined) => ['relay-subkeys', 'project', projectId, 'usage'] as const,
};

function getProductById(productId: string | undefined) {
  return mockRelayProducts.find((product) => product.id === productId);
}

function getKeyById(keyId: string | undefined) {
  return mockRelayKeys.find((key) => key.id === keyId);
}

function getWalletByKeyId(keyId: string | undefined) {
  return mockRelayWallets.find((wallet) => wallet.relayKeyId === keyId);
}

function filterKeysByProject(projectId: string | null | undefined) {
  if (!projectId) return mockRelayKeys;
  return mockRelayKeys.filter((key) => key.projectId === projectId);
}

async function withMockFallback<T>(apiCall: () => Promise<T>, fallback: T): Promise<T> {
  if (!relayApiEnabled()) return fallback;
  return apiCall();
}

async function listRelayProducts(): Promise<RelayProduct[]> {
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>('/products');
    return requireArrayResponse<RelayProduct>(response, ['products', 'relayProducts'], 'products');
  }, mockRelayProducts);
}

async function getRelayProductDetail(productId: string): Promise<RelayProduct | undefined> {
  const fallback = getProductById(productId);
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>(`/products/${encodedPath(productId)}`);
    return requireEntityResponse<RelayProduct>(response, ['product', 'relayProduct'], 'product');
  }, fallback);
}

async function listRelayKeys(): Promise<RelayKey[]> {
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>('/keys');
    return requireArrayResponse<RelayKey>(response, ['keys', 'relayKeys'], 'keys');
  }, mockRelayKeys);
}

async function getRelayKeyDetail(keyId: string): Promise<RelayKey | undefined> {
  const fallback = getKeyById(keyId);
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>(`/keys/${encodedPath(keyId)}`);
    return requireEntityResponse<RelayKey>(response, ['key', 'relayKey'], 'key');
  }, fallback);
}

async function getRelayWallet(keyId: string): Promise<RelayWallet | undefined> {
  const fallback = getWalletByKeyId(keyId);
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>(`/keys/${encodedPath(keyId)}/wallet`);
    return requireEntityResponse<RelayWallet>(response, ['wallet', 'relayWallet'], 'wallet');
  }, fallback);
}

async function listRelayLedgerEntries(keyId: string): Promise<RelayWalletLedgerEntry[]> {
  const fallback = mockRelayLedgerEntries.filter((entry) => entry.relayKeyId === keyId);
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>(`/keys/${encodedPath(keyId)}/ledger`);
    return requireArrayResponse<RelayWalletLedgerEntry>(response, ['ledgerEntries', 'relayWalletLedgerEntries'], 'ledger entries');
  }, fallback);
}

async function listRelayRequestTraces(): Promise<RelayRequestTrace[]> {
  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>('/requests');
    return requireArrayResponse<RelayRequestTrace>(response, ['requests', 'relayRequestTraces'], 'requests');
  }, mockRelayRequestTraces);
}

async function listRelayChannelPoolHealth(): Promise<RelayChannelPoolHealth[]> {
  const fallback = mockRelayProducts.map((product) => ({
    productId: product.id,
    productName: product.name,
    status: product.poolHealth,
    healthyChannels: product.channelPool.filter((channel) => channel.health === 'healthy').length,
    degradedChannels: product.channelPool.filter((channel) => channel.health === 'degraded').length,
    unavailableChannels: product.channelPool.filter((channel) => channel.health === 'unavailable').length,
    riskReason: product.channelPool.find((channel) => channel.unavailableReason)?.unavailableReason,
    channels: product.channelPool,
  }));

  return withMockFallback(async () => {
    const response = await relayApiRequest<unknown>('/channel-pool-health');
    return requireArrayResponse<RelayChannelPoolHealth>(response, ['channelPoolHealth', 'poolHealth', 'relayChannelPoolHealth'], 'channel pool health');
  }, fallback);
}

function buildProjectRelayOverview(projectId: string | null | undefined): ProjectRelayOverview {
  const keys = filterKeysByProject(projectId);
  const productIds = new Set(keys.map((key) => key.productId));
  const keyIds = new Set(keys.map((key) => key.id));

  return {
    products: mockRelayProducts.filter((product) => (projectId ? productIds.has(product.id) : product.status === 'active' || productIds.has(product.id))),
    keys,
    wallets: mockRelayWallets.filter((wallet) => keyIds.has(wallet.relayKeyId)),
    recentRequests: mockRelayRequestTraces.filter((trace) => keys.some((key) => key.name === trace.keyName)).slice(0, 5),
    usage: mockRelayUsageSummaries.filter((summary) => keyIds.has(summary.relayKeyId)),
  };
}

async function getProjectRelayOverview(projectId: string | null | undefined): Promise<ProjectRelayOverview> {
  const fallback = buildProjectRelayOverview(projectId);
  const effectiveProjectId = projectId;
  if (!effectiveProjectId) {
    if (relayApiEnabled()) throw new Error('Relay project REST requires a selected project ID');
    return fallback;
  }

  return withMockFallback(async () => {
    const response = await relayProjectApiRequest<unknown>(effectiveProjectId, '/overview');
    return requireEntityResponse<ProjectRelayOverview>(response, ['overview', 'projectRelayOverview'], 'project overview');
  }, fallback);
}

function buildProjectRelayUsage(projectId: string | null | undefined): ProjectRelayUsage {
  const keys = filterKeysByProject(projectId);
  const keyIds = new Set(keys.map((key) => key.id));
  return {
    wallets: mockRelayWallets.filter((wallet) => keyIds.has(wallet.relayKeyId)),
    ledgerEntries: mockRelayLedgerEntries.filter((entry) => keyIds.has(entry.relayKeyId)),
    usage: mockRelayUsageSummaries.filter((summary) => keyIds.has(summary.relayKeyId)),
    recentRequests: mockRelayRequestTraces.filter((trace) => keys.some((key) => key.name === trace.keyName)),
  };
}

async function getProjectRelayUsage(projectId: string | null | undefined): Promise<ProjectRelayUsage> {
  const fallback = buildProjectRelayUsage(projectId);
  const effectiveProjectId = projectId;
  if (!effectiveProjectId) {
    if (relayApiEnabled()) throw new Error('Relay project REST requires a selected project ID');
    return fallback;
  }

  return withMockFallback(async () => {
    const response = await relayProjectApiRequest<unknown>(effectiveProjectId, '/usage');
    return requireEntityResponse<ProjectRelayUsage>(response, ['usage', 'projectRelayUsage'], 'project usage');
  }, fallback);
}

export function useRelayProductsQuery() {
  return useQuery({ queryKey: relayQueryKeys.products, queryFn: listRelayProducts, staleTime: 60_000 });
}

export function useRelayProductDetailQuery(productId: string | undefined) {
  return useQuery({
    queryKey: relayQueryKeys.product(productId),
    queryFn: () => getRelayProductDetail(productId ?? ''),
    enabled: Boolean(productId),
    staleTime: 60_000,
  });
}

export function useRelayChannelPoolQuery(productId: string | undefined) {
  return useQuery({
    queryKey: relayQueryKeys.channelPool(productId),
    queryFn: async () => (await getRelayProductDetail(productId ?? ''))?.channelPool ?? [],
    enabled: Boolean(productId),
    staleTime: 60_000,
  });
}

export function useRelayKeysQuery() {
  return useQuery({ queryKey: relayQueryKeys.keys, queryFn: listRelayKeys, staleTime: 60_000 });
}

export function useRelayKeyDetailQuery(keyId: string | undefined) {
  return useQuery({ queryKey: relayQueryKeys.key(keyId), queryFn: () => getRelayKeyDetail(keyId ?? ''), enabled: Boolean(keyId), staleTime: 60_000 });
}

export function useRelayWalletQuery(keyId: string | undefined) {
  return useQuery({ queryKey: relayQueryKeys.wallet(keyId), queryFn: () => getRelayWallet(keyId ?? ''), enabled: Boolean(keyId), staleTime: 60_000 });
}

export function useRelayLedgerEntriesQuery(keyId: string | undefined) {
  return useQuery({
    queryKey: relayQueryKeys.ledger(keyId),
    queryFn: () => listRelayLedgerEntries(keyId ?? ''),
    enabled: Boolean(keyId),
    staleTime: 60_000,
  });
}

export function useRelayRequestTraceQuery() {
  return useQuery({ queryKey: relayQueryKeys.requests, queryFn: listRelayRequestTraces, staleTime: 30_000 });
}

export function useRelayChannelPoolHealthQuery() {
  return useQuery({ queryKey: relayQueryKeys.poolHealth, queryFn: listRelayChannelPoolHealth, staleTime: 30_000 });
}

export function useProjectRelayOverviewQuery(projectId?: string | null) {
  const selectedProjectId = useSelectedProjectId();
  const effectiveProjectId = projectId ?? selectedProjectId;
  return useQuery({
    queryKey: relayQueryKeys.projectOverview(effectiveProjectId),
    queryFn: () => getProjectRelayOverview(effectiveProjectId),
    staleTime: 60_000,
  });
}

export function useProjectRelayProductsQuery(projectId?: string | null) {
  const selectedProjectId = useSelectedProjectId();
  const effectiveProjectId = projectId ?? selectedProjectId;
  return useQuery({
    queryKey: relayQueryKeys.projectProducts(effectiveProjectId),
    queryFn: async () => (await getProjectRelayOverview(effectiveProjectId)).products,
    staleTime: 60_000,
  });
}

export function useProjectRelayKeysQuery(projectId?: string | null) {
  const selectedProjectId = useSelectedProjectId();
  const effectiveProjectId = projectId ?? selectedProjectId;
  return useQuery({
    queryKey: relayQueryKeys.projectKeys(effectiveProjectId),
    queryFn: async () => (await getProjectRelayOverview(effectiveProjectId)).keys,
    staleTime: 60_000,
  });
}

export function useProjectRelayUsageQuery(projectId?: string | null) {
  const selectedProjectId = useSelectedProjectId();
  const effectiveProjectId = projectId ?? selectedProjectId;
  return useQuery({
    queryKey: relayQueryKeys.projectUsage(effectiveProjectId),
    queryFn: () => getProjectRelayUsage(effectiveProjectId),
    staleTime: 60_000,
  });
}

export function useCreateRelayProductMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRelayProductInput) => {
      const fallback: RelayProduct = {
        ...mockRelayProducts[0],
        ...input,
        id: `prod-preview-${Date.now()}`,
        status: 'draft',
        channelPool: [],
        keyCount: 0,
        activeKeyCount: 0,
        monthlyRequestCount: 0,
        monthlyTokenCount: 0,
        monthlyCost: 0,
        poolHealth: 'unavailable',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>('/products', { method: 'POST', body: input });
        return requireEntityResponse<RelayProduct>(response, ['product', 'relayProduct'], 'created product');
      }, fallback);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: relayQueryKeys.products }),
  });
}

export function useUpdateRelayProductMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: UpdateRelayProductInput }) => {
      const fallback: RelayProduct = { ...(getProductById(id) ?? mockRelayProducts[0]), ...input, updatedAt: new Date().toISOString() };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>(`/products/${encodedPath(id)}`, { method: 'PATCH', body: input });
        return requireEntityResponse<RelayProduct>(response, ['product', 'relayProduct'], 'updated product');
      }, fallback);
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.products });
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.product(variables.id) });
    },
  });
}

function resolveBindingProductId(binding: RelayProductChannel | undefined, productId: string | undefined) {
  return productId ?? binding?.productId;
}

function invalidateRelayProductChannelQueries(queryClient: ReturnType<typeof useQueryClient>, productId: string | undefined) {
  if (productId) {
    queryClient.invalidateQueries({ queryKey: relayQueryKeys.product(productId) });
    queryClient.invalidateQueries({ queryKey: relayQueryKeys.channelPool(productId) });
  }
  queryClient.invalidateQueries({ queryKey: relayQueryKeys.products });
  queryClient.invalidateQueries({ queryKey: relayQueryKeys.poolHealth });
}

export function useBindRelayChannelMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: BindRelayChannelInput) => {
      const fallback: RelayProductChannel = {
        id: `bind-preview-${Date.now()}`,
        productId: input.productId,
        channelId: input.channelId,
        channelName: `Channel ${input.channelId}`,
        provider: 'openai_compatible',
        priority: input.priority,
        weight: input.weight,
        status: 'active',
        health: 'healthy',
        allowFallback: input.allowFallback,
        modelFilter: input.modelFilter,
        quotaRemainingPercent: 100,
        errorRatePercent: 0,
        latencyMs: 0,
        lastCheckedAt: new Date().toISOString(),
      };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>('/product-channels', { method: 'POST', body: input });
        return requireEntityResponse<RelayProductChannel>(response, ['channel', 'binding', 'productChannel', 'relayProductChannel'], 'created product channel');
      }, fallback);
    },
    onSuccess: (_data, variables) => invalidateRelayProductChannelQueries(queryClient, variables.productId),
  });
}

export function useUpdateRelayChannelBindingMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, productId, ...input }: UpdateRelayChannelBindingInput) => {
      const fallbackProduct = productId ? getProductById(productId) : undefined;
      const fallback = fallbackProduct?.channelPool.find((channel) => channel.id === id);
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>(`/product-channels/${encodedPath(id)}`, { method: 'PATCH', body: input });
        return requireEntityResponse<RelayProductChannel>(response, ['channel', 'binding', 'productChannel', 'relayProductChannel'], 'updated product channel');
      }, fallback);
    },
    onSuccess: (data, variables) => invalidateRelayProductChannelQueries(queryClient, resolveBindingProductId(data, variables.productId)),
  });
}

export function useDeleteRelayChannelBindingMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, productId }: { id: string; productId?: string }) => {
      return withMockFallback(async () => {
        await relayApiRequest<unknown>(`/product-channels/${encodedPath(id)}`, { method: 'DELETE' });
        return { id, productId };
      }, { id, productId });
    },
    onSuccess: (_data, variables) => invalidateRelayProductChannelQueries(queryClient, variables.productId),
  });
}

export function useCreateRelayKeyMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRelayKeyInput) => {
      const fallback: RelayKey = {
        ...mockRelayKeys[0],
        id: `rkey-preview-${Date.now()}`,
        projectId: input.projectId,
        productId: input.productId,
        productName: getProductById(input.productId)?.name ?? 'Relay product',
        name: input.name,
        status: 'active',
        derivedStates: [],
        balanceMode: input.balanceMode,
        expiresAt: input.expiresAt,
        limits: input.limits,
        usage: { todayRequests: 0, todayTokens: 0, monthlyCost: 0 },
        createdAt: new Date().toISOString(),
        maskedKey: 'ahub_sk_live_new...once',
      };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>('/keys', { method: 'POST', body: input });
        return requireEntityResponse<RelayKey>(response, ['key', 'relayKey'], 'created key');
      }, fallback);
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.keys });
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.projectOverview(variables.projectId) });
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.projectKeys(variables.projectId) });
    },
  });
}

export function useSuspendRelayKeyMutation() {
  return useRelayKeyStatusMutation('suspended');
}

export function useResumeRelayKeyMutation() {
  return useRelayKeyStatusMutation('active');
}

export function useArchiveRelayKeyMutation() {
  return useRelayKeyStatusMutation('archived');
}

function useRelayKeyStatusMutation(status: RelayKeyStatus) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, note }: { id: string; note?: string }) => {
      const fallback: RelayKey = { ...(getKeyById(id) ?? mockRelayKeys[0]), status };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>(`/keys/${encodedPath(id)}/status`, { method: 'PATCH', body: { status, note } });
        return requireEntityResponse<RelayKey>(response, ['key', 'relayKey'], 'updated key status');
      }, fallback);
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.keys });
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.key(variables.id) });
    },
  });
}

export function useRechargeRelayWalletMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: RechargeRelayWalletInput) => {
      const fallback: RelayWalletLedgerEntry = {
        id: `ledger-preview-${Date.now()}`,
        relayKeyId: input.relayKeyId,
        type: 'recharge',
        amount: input.amount,
        currency: 'USD',
        balanceAfter: (getWalletByKeyId(input.relayKeyId)?.availableAmount ?? 0) + input.amount,
        operator: 'current-user',
        note: input.note,
        createdAt: new Date().toISOString(),
      };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>('/wallets/recharge', { method: 'POST', body: input });
        return requireEntityResponse<RelayWalletLedgerEntry>(response, ['ledgerEntry', 'relayWalletLedgerEntry'], 'wallet recharge ledger entry');
      }, fallback);
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.wallet(variables.relayKeyId) });
      queryClient.invalidateQueries({ queryKey: relayQueryKeys.ledger(variables.relayKeyId) });
    },
  });
}

export function useAdjustRelayKeyLimitMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, limits }: { id: string; limits: RelayKeyLimitSnapshot }) => {
      const fallback: RelayKey = { ...(getKeyById(id) ?? mockRelayKeys[0]), limits };
      return withMockFallback(async () => {
        const response = await relayApiRequest<unknown>(`/keys/${encodedPath(id)}/limits`, { method: 'PATCH', body: { limits } });
        return requireEntityResponse<RelayKey>(response, ['key', 'relayKey'], 'updated key limits');
      }, fallback);
    },
    onSuccess: (_data, variables) => queryClient.invalidateQueries({ queryKey: relayQueryKeys.key(variables.id) }),
  });
}
