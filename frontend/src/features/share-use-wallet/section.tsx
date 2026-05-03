import { useState } from 'react';
import { ShareUseWalletPanel } from './panel';
import { useShareUseLedgerQuery, useShareUseWalletQuery } from './data';
import { ShareUseLedgerTable } from './table';

export function ShareUseWalletSection({ enabled }: { enabled: boolean }) {
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const walletQuery = useShareUseWalletQuery(enabled);
  const ledgerQuery = useShareUseLedgerQuery(page, pageSize, enabled);

  return (
    <div className='space-y-6'>
      <ShareUseWalletPanel wallet={walletQuery.data} isLoading={walletQuery.isLoading} error={walletQuery.error} />
      <div className='space-y-2'>
        <div className='text-sm font-medium'>Settlement ledger</div>
        <ShareUseLedgerTable
          ledgerPage={ledgerQuery.data}
          isLoading={ledgerQuery.isLoading}
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
