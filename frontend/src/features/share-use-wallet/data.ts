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

export interface ShareUseLedgerPage {
  entries: ShareUsePointLedgerEntry[];
  ledgerEntries: ShareUsePointLedgerEntry[];
  totalCount: number;
  page: number;
  pageSize: number;
  hasNext: boolean;
  hasPrev: boolean;
}

async function getShareUseWallet(): Promise<ShareUsePointWallet | null | undefined> {
  const response = await apiRequest<{ wallet?: ShareUsePointWallet | null; data?: ShareUsePointWallet | null }>('/admin/share-use/wallet', {
    requireAuth: true,
  });
  return response.wallet ?? response.data;
}

async function fetchShareUseLedgerPage(page: number, pageSize: number): Promise<ShareUseLedgerPage> {
  const searchParams = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
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
    ledgerEntries: data?.ledgerEntries ?? entries,
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

export function useShareUseLedgerQuery(page: number, pageSize: number, enabled = true) {
  return useQuery<ShareUseLedgerPage, Error, ShareUseLedgerPage, readonly ['share-use-ledger', number, number]>({
    queryKey: ['share-use-ledger', page, pageSize],
    queryFn: ({ queryKey }) => {
      const [, currentPage, currentPageSize] = queryKey;
      return fetchShareUseLedgerPage(currentPage, currentPageSize);
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
