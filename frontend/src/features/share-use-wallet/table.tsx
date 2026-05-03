import { useMemo } from 'react';
import { X } from 'lucide-react';
import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { DateRangePicker } from '@/components/date-range-picker';
import { ServerSidePagination } from '@/components/server-side-pagination';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { TableSkeleton } from '@/components/ui/table-skeleton';
import { pageInfoSchema } from '@/gql/pagination';
import type { DateTimeRangeValue } from '@/utils/date-range';
import type { ShareUseLedgerPage, ShareUsePointLedgerEntry } from './data';

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(value);
}

const sceneFilterOptions = [
  { value: 'all', label: 'All scenes' },
  { value: 'contribution_pending', label: 'Contribution pending' },
  { value: 'contribution_reward', label: 'Contribution reward' },
  { value: 'consume', label: 'Consume' },
  { value: 'adjustment', label: 'Adjustment' },
] as const;

const directionFilterOptions = [
  { value: 'all', label: 'All directions' },
  { value: 'credit', label: 'Credit' },
  { value: 'debit', label: 'Debit' },
] as const;

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
  ledgerPage?: ShareUseLedgerPage;
  isLoading: boolean;
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
  const entries = ledgerPage?.entries ?? ledgerPage?.ledgerEntries ?? [];
  const hasFilters = Boolean(scene || direction || dateRange?.from || dateRange?.to);
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
      <div className='mb-4 flex flex-wrap items-center gap-2'>
        <Select value={scene ?? 'all'} onValueChange={(value) => onSceneChange(value === 'all' ? undefined : value)}>
          <SelectTrigger size='sm' className='w-[220px]'>
            <SelectValue placeholder='All scenes' />
          </SelectTrigger>
          <SelectContent>
            {sceneFilterOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={direction ?? 'all'} onValueChange={(value) => onDirectionChange(value === 'all' ? undefined : value)}>
          <SelectTrigger size='sm' className='w-[180px]'>
            <SelectValue placeholder='All directions' />
          </SelectTrigger>
          <SelectContent>
            {directionFilterOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <DateRangePicker value={dateRange} onChange={onDateRangeChange} />
        {hasFilters && (
          <Button variant='ghost' size='sm' className='h-8 px-2 lg:px-3' onClick={onResetFilters}>
            Reset filters
            <X className='ml-2 h-4 w-4' />
          </Button>
        )}
      </div>
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
                  {hasFilters ? 'No point ledger entries match the current filters.' : 'No point ledger entries yet.'}
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
