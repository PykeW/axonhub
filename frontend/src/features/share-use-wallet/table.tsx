import { useMemo } from 'react';
import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { TableSkeleton } from '@/components/ui/table-skeleton';
import { ServerSidePagination } from '@/components/server-side-pagination';
import { pageInfoSchema } from '@/gql/pagination';
import type { ShareUseLedgerPage, ShareUsePointLedgerEntry } from './data';

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(value);
}

const columns: ColumnDef<ShareUsePointLedgerEntry>[] = [
  {
    accessorKey: 'createdAt',
    header: 'Time',
    cell: ({ row }) => <span className='text-xs'>{row.original.createdAt}</span>,
  },
  {
    accessorKey: 'scene',
    header: 'Scene',
    cell: ({ row }) => <span className='font-medium'>{row.original.scene}</span>,
  },
  {
    accessorKey: 'direction',
    header: 'Direction',
  },
  {
    accessorKey: 'points',
    header: 'Points',
    cell: ({ row }) => (
      <span>
        {row.original.direction === 'credit' ? '+' : '-'}
        {formatNumber(row.original.points)}
      </span>
    ),
  },
  {
    accessorKey: 'balanceAfter',
    header: 'Balance after',
    cell: ({ row }) => <span>{formatNumber(row.original.balanceAfter)}</span>,
  },
  {
    accessorKey: 'remark',
    header: 'Remark',
    cell: ({ row }) => <span className='text-xs text-muted-foreground'>{row.original.remark ?? '-'}</span>,
  },
];

export function ShareUseLedgerTable({
  ledgerPage,
  isLoading,
  onNextPage,
  onPreviousPage,
  onFirstPage,
  onPageSizeChange,
}: {
  ledgerPage?: ShareUseLedgerPage;
  isLoading: boolean;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onFirstPage: () => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const entries = ledgerPage?.entries ?? ledgerPage?.ledgerEntries ?? [];
  const pageInfo = useMemo(
    () =>
      pageInfoSchema.parse({
        hasPreviousPage: ledgerPage?.hasPrev ?? false,
        hasNextPage: ledgerPage?.hasNext ?? false,
        startCursor: ledgerPage ? String(ledgerPage.page) : null,
        endCursor: ledgerPage ? String(ledgerPage.page) : null,
      }),
    [ledgerPage]
  );

  const table = useReactTable({
    data: entries,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  });

  return (
    <div className='flex flex-1 flex-col overflow-hidden'>
      <div className='shadow-soft relative flex-1 overflow-auto overflow-x-hidden rounded-2xl border border-[var(--table-border)]'>
        <Table className='border-separate border-spacing-0 rounded-2xl bg-[var(--table-background)]'>
          <TableHeader className='sticky top-0 z-20 bg-[var(--table-header)] shadow-sm'>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className='group/row border-0'>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} className='text-muted-foreground border-0 text-xs font-semibold uppercase'>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody className='space-y-1 !bg-[var(--table-background)] p-2'>
            {isLoading ? (
              <TableSkeleton rows={ledgerPage?.pageSize ?? 10} columns={columns.length} />
            ) : table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} className='group/row rounded-xl border-0 !bg-[var(--table-background)]'>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className='border-0 bg-inherit px-4 py-3'>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow className='!bg-[var(--table-background)]'>
                <TableCell colSpan={columns.length} className='h-24 !bg-[var(--table-background)] text-center'>
                  No point ledger entries yet.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className='mt-4 flex-shrink-0'>
        <ServerSidePagination
          pageInfo={pageInfo}
          pageSize={ledgerPage?.pageSize ?? 10}
          dataLength={entries.length}
          totalCount={ledgerPage?.totalCount ?? entries.length}
          selectedRows={0}
          onNextPage={onNextPage}
          onPreviousPage={onPreviousPage}
          onFirstPage={onFirstPage}
          onPageSizeChange={onPageSizeChange}
        />
      </div>
    </div>
  );
}
