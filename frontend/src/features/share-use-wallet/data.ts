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

async function getShareUseWallet(): Promise<ShareUsePointWallet | null | undefined> {
  const response = await apiRequest<{ wallet?: ShareUsePointWallet | null; data?: ShareUsePointWallet | null }>('/admin/share-use/wallet', {
    requireAuth: true,
  });
  return response.wallet ?? response.data;
}

async function listShareUseLedger(): Promise<ShareUsePointLedgerEntry[]> {
  const response = await apiRequest<{ ledgerEntries?: ShareUsePointLedgerEntry[]; data?: ShareUsePointLedgerEntry[] }>(
    '/admin/share-use/ledger',
    {
      requireAuth: true,
    }
  );
  return response.ledgerEntries ?? response.data ?? [];
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

export function useShareUseLedgerQuery(enabled = true) {
  return useQuery({
    queryKey: ['share-use-ledger'],
    queryFn: listShareUseLedger,
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
