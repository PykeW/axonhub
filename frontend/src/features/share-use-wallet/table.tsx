import { useMemo, useState } from 'react';
import { type ColumnDef, type PaginationState, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { TableSkeleton } from '@/components/ui/table-skeleton';
import { ServerSidePagination } from '@/components/server-side-pagination';
import { pageInfoSchema, type PageInfo } from '@/gql/pagination';
import type { ShareUsePointLedgerEntry } from './data';

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(value);
}

function buildPageInfo(pageIndex: number, pageSize: number, totalCount: number): PageInfo {
  const pageCount = Math.max(1, Math.ceil(totalCount / pageSize));
  return pageInfoSchema.parse({
    hasPreviousPage: pageIndex > 0,
    hasNextPage: pageIndex < pageCount - 1,
    startCursor: pageCount > 0 ? String(pageIndex) : null,
    endCursor: pageCount > 0 ? String(pageIndex) : null,
  });
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
  entries,
  isLoading,
}: {
  entries: ShareUsePointLedgerEntry[];
  isLoading: boolean;
}) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });

  const pagedEntries = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize;
    return entries.slice(start, start + pagination.pageSize);
  }, [entries, pagination.pageIndex, pagination.pageSize]);

  const pageInfo = useMemo(() => buildPageInfo(pagination.pageIndex, pagination.pageSize, entries.length), [entries.length, pagination.pageIndex, pagination.pageSize]);

  const table = useReactTable({
    data: pagedEntries,
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
              <TableSkeleton rows={pagination.pageSize} columns={columns.length} />
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
          pageSize={pagination.pageSize}
          dataLength={pagedEntries.length}
          totalCount={entries.length}
          selectedRows={0}
          onNextPage={() => setPagination((prev) => ({ ...prev, pageIndex: prev.pageIndex + 1 }))}
          onPreviousPage={() => setPagination((prev) => ({ ...prev, pageIndex: Math.max(0, prev.pageIndex - 1) }))}
          onFirstPage={() => setPagination((prev) => ({ ...prev, pageIndex: 0 }))}
          onPageSizeChange={(pageSize) => setPagination({ pageIndex: 0, pageSize })}
        />
      </div>
    </div>
  );
}
