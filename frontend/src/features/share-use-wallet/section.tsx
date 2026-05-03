import { useMemo } from 'react';
import { buildDateRangeWhereClause, type DateTimeRangeValue } from '@/utils/date-range';
import { ShareUseWalletPanel } from './panel';
import { type ShareUseLedgerFilters, useShareUseLedgerQuery, useShareUseWalletQuery } from './data';
import { ShareUseLedgerTable } from './table';

export function ShareUseWalletSection({
  enabled,
  page,
  pageSize,
  scene,
  direction,
  dateRange,
  onSceneChange,
  onDirectionChange,
  onDateRangeChange,
  onResetFilters,
  onNextPage,
  onPreviousPage,
  onFirstPage,
  onPageSizeChange,
}: {
  enabled: boolean;
  page: number;
  pageSize: number;
  scene?: string;
  direction?: string;
  dateRange?: DateTimeRangeValue;
  onSceneChange: (scene?: string) => void;
  onDirectionChange: (direction?: string) => void;
  onDateRangeChange: (range: DateTimeRangeValue | undefined) => void;
  onResetFilters: () => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onFirstPage: () => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const walletQuery = useShareUseWalletQuery(enabled);
  const ledgerFilters = useMemo<ShareUseLedgerFilters>(
    () => ({
      scene,
      direction,
      ...buildDateRangeWhereClause(dateRange),
    }),
    [scene, direction, dateRange]
  );
  const ledgerQuery = useShareUseLedgerQuery(page, pageSize, ledgerFilters, enabled);

  return (
    <div className='space-y-6'>
      <ShareUseWalletPanel wallet={walletQuery.data} isLoading={walletQuery.isLoading} error={walletQuery.error} />
      <div className='space-y-2'>
        <div className='text-sm font-medium'>Settlement ledger</div>
        <ShareUseLedgerTable
          ledgerPage={ledgerQuery.data}
          isLoading={ledgerQuery.isLoading}
          scene={scene}
          direction={direction}
          dateRange={dateRange}
          onSceneChange={onSceneChange}
          onDirectionChange={onDirectionChange}
          onDateRangeChange={onDateRangeChange}
          onResetFilters={onResetFilters}
          onNextPage={onNextPage}
          onPreviousPage={onPreviousPage}
          onFirstPage={onFirstPage}
          onPageSizeChange={onPageSizeChange}
        />
      </div>
    </div>
  );
}
