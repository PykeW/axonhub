import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import type { ShareUsePointWallet } from './data';

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(value);
}

export function ShareUseWalletPanel({ wallet, isLoading, error }: { wallet?: ShareUsePointWallet | null; isLoading: boolean; error: unknown }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Share/Use points</CardTitle>
        <CardDescription>Successful cross-user shared-channel settlement writes contribution and consume records here.</CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {isLoading ? <Skeleton className='h-40 w-full rounded-md' /> : null}
        {error ? (
          <Alert>
            <AlertTitle>Share/Use wallet unavailable</AlertTitle>
            <AlertDescription>{error instanceof Error ? error.message : 'Unknown error'}</AlertDescription>
          </Alert>
        ) : null}
        {!isLoading && !error ? (
          <div className='grid gap-3 sm:grid-cols-2'>
            <div className='bg-muted/40 rounded-lg p-3'>Available points: {formatNumber(wallet?.availablePoints ?? 0)}</div>
            <div className='bg-muted/40 rounded-lg p-3'>Pending points: {formatNumber(wallet?.pendingPoints ?? 0)}</div>
            <div className='bg-muted/40 rounded-lg p-3'>Frozen points: {formatNumber(wallet?.frozenPoints ?? 0)}</div>
            <div className='bg-muted/40 rounded-lg p-3'>
              Lifetime earned / spent: {formatNumber(wallet?.lifetimeEarned ?? 0)} / {formatNumber(wallet?.lifetimeSpent ?? 0)}
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
