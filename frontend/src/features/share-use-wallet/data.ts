import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '@/lib/api-client';

export interface ShareUsePointWallet {
  id: string;
  userId: string;
  availablePoints: number;
  pendingPoints: number;
  frozenPoints: number;
  lifetimeEarned: number;
  lifetimeSpent: number;
  updatedAt: string;
}

export interface ShareUsePointLedgerEntry {
  id: string;
  userId: string;
  direction: 'credit' | 'debit' | string;
  scene: 'contribution_pending' | 'contribution_reward' | 'consume' | 'adjustment' | string;
  points: number;
  balanceBefore: number;
  balanceAfter: number;
  relatedChannelId?: string;
  relatedRequestId?: string;
  relatedUsageLogId?: string;
  relatedApiKeyId?: string;
  relatedProjectId?: string;
  conversionRateSnapshot?: string;
  settlementStatus: string;
  remark?: string;
  createdAt: string;
}

export interface ShareUseWalletUsage {
  wallet?: ShareUsePointWallet | null;
  ledgerEntries: ShareUsePointLedgerEntry[];
}

export interface ShareUseLedgerFilters {
  scene?: ShareUsePointLedgerEntry['scene'];
  direction?: ShareUsePointLedgerEntry['direction'];
  createdAtGTE?: string;
  createdAtLTE?: string;
}

export interface ShareUseLedgerPage {
  entries: ShareUsePointLedgerEntry[];
  ledgerEntries: ShareUsePointLedgerEntry[];
  totalCount: number;
  page: number;
  pageSize: number;
  hasNext: boolean;
  hasPrev: boolean;
}

type ShareUseLedgerQueryKey = readonly ['share-use-ledger', number, number, string, string, string, string];

async function getShareUseWallet(): Promise<ShareUsePointWallet | null | undefined> {
  const response = await apiRequest<{ wallet?: ShareUsePointWallet | null; data?: ShareUsePointWallet | null }>('/admin/share-use/wallet', {
    requireAuth: true,
  });
  return response.wallet ?? response.data;
}

async function fetchShareUseLedgerPage(page: number, pageSize: number, filters: ShareUseLedgerFilters): Promise<ShareUseLedgerPage> {
  const searchParams = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
  if (filters.scene) {
    searchParams.set('scene', filters.scene);
  }
  if (filters.direction) {
    searchParams.set('direction', filters.direction);
  }
  if (filters.createdAtGTE) {
    searchParams.set('createdAtGTE', filters.createdAtGTE);
  }
  if (filters.createdAtLTE) {
    searchParams.set('createdAtLTE', filters.createdAtLTE);
  }

  const response = await apiRequest<{
    entries?: ShareUsePointLedgerEntry[];
    ledgerEntries?: ShareUsePointLedgerEntry[];
    totalCount?: number;
    page?: number;
    pageSize?: number;
    hasNext?: boolean;
    hasPrev?: boolean;
    data?: ShareUseLedgerPage;
  }>(`/admin/share-use/ledger?${searchParams.toString()}`, {
    requireAuth: true,
  });
  const data = response.data;
  const entries = response.entries ?? response.ledgerEntries ?? data?.entries ?? data?.ledgerEntries ?? [];
  return {
    entries,
    ledgerEntries: response.ledgerEntries ?? data?.ledgerEntries ?? entries,
    totalCount: response.totalCount ?? data?.totalCount ?? entries.length,
    page: response.page ?? data?.page ?? page,
    pageSize: response.pageSize ?? data?.pageSize ?? pageSize,
    hasNext: response.hasNext ?? data?.hasNext ?? false,
    hasPrev: response.hasPrev ?? data?.hasPrev ?? page > 0,
  };
}

async function getShareUseUsage(): Promise<ShareUseWalletUsage> {
  const response = await apiRequest<{ usage?: ShareUseWalletUsage; data?: ShareUseWalletUsage }>('/admin/share-use/usage', {
    requireAuth: true,
  });
  return response.usage ?? response.data ?? { wallet: null, ledgerEntries: [] };
}

export function useShareUseWalletQuery(enabled = true) {
  return useQuery({
    queryKey: ['share-use-wallet'],
    queryFn: getShareUseWallet,
    enabled,
    staleTime: 60_000,
  });
}

export function useShareUseLedgerQuery(page: number, pageSize: number, filters: ShareUseLedgerFilters, enabled = true) {
  return useQuery<ShareUseLedgerPage, Error, ShareUseLedgerPage, ShareUseLedgerQueryKey>({
    queryKey: [
      'share-use-ledger',
      page,
      pageSize,
      filters.scene ?? '',
      filters.direction ?? '',
      filters.createdAtGTE ?? '',
      filters.createdAtLTE ?? '',
    ],
    queryFn: ({ queryKey }) => {
      const [, currentPage, currentPageSize, scene, direction, createdAtGTE, createdAtLTE] = queryKey;
      return fetchShareUseLedgerPage(currentPage, currentPageSize, {
        scene: scene || undefined,
        direction: direction || undefined,
        createdAtGTE: createdAtGTE || undefined,
        createdAtLTE: createdAtLTE || undefined,
      });
    },
    enabled,
    staleTime: 60_000,
  });
}

export function useShareUseUsageQuery(enabled = true) {
  return useQuery({
    queryKey: ['share-use-usage'],
    queryFn: getShareUseUsage,
    enabled,
    staleTime: 60_000,
  });
}
