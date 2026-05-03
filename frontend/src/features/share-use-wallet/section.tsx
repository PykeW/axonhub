import { useMemo, useState } from 'react';
import { buildDateRangeWhereClause, type DateTimeRangeValue } from '@/utils/date-range';
import { ShareUseWalletPanel } from './panel';
import { type ShareUseLedgerFilters, useShareUseLedgerQuery, useShareUseWalletQuery } from './data';
import { ShareUseLedgerTable } from './table';

export function ShareUseWalletSection({ enabled }: { enabled: boolean }) {
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [scene, setScene] = useState<string | undefined>();
  const [direction, setDirection] = useState<string | undefined>();
  const [dateRange, setDateRange] = useState<DateTimeRangeValue | undefined>();

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
          onSceneChange={(nextScene) => {
            setPage(0);
            setScene(nextScene);
          }}
          onDirectionChange={(nextDirection) => {
            setPage(0);
            setDirection(nextDirection);
          }}
          onDateRangeChange={(nextDateRange) => {
            setPage(0);
            setDateRange(nextDateRange);
          }}
          onResetFilters={() => {
            setPage(0);
            setScene(undefined);
            setDirection(undefined);
            setDateRange(undefined);
          }}
          onNextPage={() => setPage((prev) => prev + 1)}
          onPreviousPage={() => setPage((prev) => Math.max(0, prev - 1))}
          onFirstPage={() => setPage(0)}
          onPageSizeChange={(nextPageSize) => {
            setPage(0);
            setPageSize(nextPageSize);
          }}
        />
      </div>
    </div>
  );
}
