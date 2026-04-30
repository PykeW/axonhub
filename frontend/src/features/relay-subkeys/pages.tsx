import { type FormEvent, type ReactNode, useMemo, useState } from 'react';
import { Outlet } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';
import { Header } from '@/components/layout/header';
import { useRoutePermissions } from '@/hooks/useRoutePermissions';
import { Main } from '@/components/layout/main';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  type RelayDerivedState,
  type RelayFailureStage,
  type RelayHealthStatus,
  type RelayKey,
  type RelayKeyStatus,
  type RelayProduct,
  type RelayProductStatus,
  type RelayProviderType,
  type RelayRequestTrace,
  type RelayWalletLedgerType,
  useAdjustRelayKeyLimitMutation,
  useArchiveRelayKeyMutation,
  useBindRelayChannelMutation,
  useDeleteRelayChannelBindingMutation,
  useCreateRelayKeyMutation,
  useCreateRelayProductMutation,
  useRechargeRelayWalletMutation,
  useRelayChannelPoolHealthQuery,
  useRelayChannelPoolQuery,
  useRelayKeyDetailQuery,
  useRelayKeysQuery,
  useRelayLedgerEntriesQuery,
  useRelayProductDetailQuery,
  useRelayProductsQuery,
  useRelayRequestTraceQuery,
  useRelayWalletQuery,
  useResumeRelayKeyMutation,
  useSuspendRelayKeyMutation,
  useUpdateRelayChannelBindingMutation,
  useUpdateRelayProductMutation,
} from './data';

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

interface DetailPageProps {
  productId?: string;
  keyId?: string;
}

const operatorNav = [
  { labelKey: 'relaySubkeys.nav.overview', fallback: 'Overview', href: '/relay-subkeys' },
  { labelKey: 'relaySubkeys.nav.products', fallback: 'Products', href: '/relay-subkeys/products' },
  { labelKey: 'relaySubkeys.nav.keys', fallback: 'Sub-Keys', href: '/relay-subkeys/keys' },
  { labelKey: 'relaySubkeys.nav.requests', fallback: 'Request trace', href: '/relay-subkeys/requests' },
  { labelKey: 'relaySubkeys.nav.poolHealth', fallback: 'Channel pool health', href: '/relay-subkeys/channel-pool-health' },
];

function useRelayText() {
  const { t } = useTranslation();
  return (key: string, fallback: string) => t(key, { defaultValue: fallback });
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US').format(value);
}

function formatCurrency(value: number, currency = 'USD') {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(value);
}

function formatDateTime(value?: string) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

function productStatusVariant(status: RelayProductStatus): BadgeVariant {
  if (status === 'active') return 'default';
  if (status === 'archived') return 'outline';
  return 'secondary';
}

function keyStatusVariant(status: RelayKeyStatus): BadgeVariant {
  if (status === 'active') return 'default';
  if (status === 'exhausted') return 'destructive';
  if (status === 'suspended') return 'secondary';
  return 'outline';
}

function healthVariant(health: RelayHealthStatus): BadgeVariant {
  if (health === 'healthy') return 'default';
  if (health === 'degraded') return 'secondary';
  return 'destructive';
}

function failureStageVariant(stage: RelayFailureStage): BadgeVariant {
  if (stage === 'none') return 'default';
  if (stage === 'settlement') return 'secondary';
  return 'destructive';
}

function ledgerTypeVariant(type: RelayWalletLedgerType): BadgeVariant {
  if (type === 'recharge' || type === 'refund' || type === 'unfreeze') return 'default';
  if (type === 'charge' || type === 'freeze') return 'secondary';
  return 'outline';
}

function StatusBadge({ children, variant }: { children: string; variant: BadgeVariant }) {
  return <Badge variant={variant}>{children}</Badge>;
}

function HealthBadge({ health }: { health: RelayHealthStatus }) {
  return <StatusBadge variant={healthVariant(health)}>{health}</StatusBadge>;
}

function ProductStatusBadge({ status }: { status: RelayProductStatus }) {
  return <StatusBadge variant={productStatusVariant(status)}>{status}</StatusBadge>;
}

function KeyStatusBadge({ status }: { status: RelayKeyStatus }) {
  return <StatusBadge variant={keyStatusVariant(status)}>{status}</StatusBadge>;
}

function DerivedStateBadges({ states }: { states: RelayDerivedState[] }) {
  if (states.length === 0) return <Badge variant='outline'>no derived risk</Badge>;
  return (
    <div className='flex flex-wrap gap-1'>
      {states.map((state) => (
        <Badge key={state} variant={state === 'low_balance' || state === 'upstream_pool_degraded' ? 'secondary' : 'destructive'}>
          {state.replaceAll('_', ' ')}
        </Badge>
      ))}
    </div>
  );
}

function PageHeader({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return (
    <div className='flex flex-col gap-3 md:flex-row md:items-start md:justify-between'>
      <div className='space-y-1'>
        <h3 className='text-lg font-semibold tracking-tight'>{title}</h3>
        <p className='text-sm text-muted-foreground'>{description}</p>
      </div>
      {actions ? <div className='flex flex-wrap gap-2'>{actions}</div> : null}
    </div>
  );
}

function MetricCard({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <Card>
      <CardHeader className='gap-1'>
        <CardDescription>{label}</CardDescription>
        <CardTitle className='text-2xl'>{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className='text-sm text-muted-foreground'>{hint}</p>
      </CardContent>
    </Card>
  );
}

function LoadingCards({ count = 4 }: { count?: number }) {
  return (
    <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
      {Array.from({ length: count }).map((_, index) => (
        <Card key={index}>
          <CardHeader>
            <Skeleton className='h-4 w-28' />
            <Skeleton className='h-7 w-20' />
          </CardHeader>
          <CardContent>
            <Skeleton className='h-4 w-full' />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <Card>
      <CardContent className='py-10 text-center'>
        <p className='font-medium'>{title}</p>
        <p className='mt-1 text-sm text-muted-foreground'>{description}</p>
      </CardContent>
    </Card>
  );
}

function ErrorState({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : 'Unknown error';
  return (
    <Alert variant='destructive'>
      <AlertTitle>Unable to load relay data</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function TableFrame({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className='overflow-x-auto'>{children}</div>
      </CardContent>
    </Card>
  );
}

function OperatorNav() {
  const tt = useRelayText();
  return (
    <div className='flex flex-wrap gap-2'>
      {operatorNav.map((item) => (
        <Button key={item.href} variant='outline' size='sm' asChild>
          <a href={item.href}>{tt(item.labelKey, item.fallback)}</a>
        </Button>
      ))}
    </div>
  );
}

function ProductTable({ products }: { products: RelayProduct[] }) {
  if (products.length === 0) {
    return <EmptyState title='No relay products' description='Create a product to bind upstream channel pools and issue keys.' />;
  }

  return (
    <TableFrame title='Product inventory' description='Operational state, pool health, and issued-key coverage for shared-capacity products.'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Product</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Pool</TableHead>
            <TableHead>Models</TableHead>
            <TableHead>Keys</TableHead>
            <TableHead className='text-right'>Monthly cost</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {products.map((product) => (
            <TableRow key={product.id}>
              <TableCell className='whitespace-normal'>
                <div className='font-medium'>{product.name}</div>
                <div className='text-xs text-muted-foreground'>{product.code}</div>
              </TableCell>
              <TableCell>
                <ProductStatusBadge status={product.status} />
              </TableCell>
              <TableCell>
                <div className='space-y-1'>
                  <HealthBadge health={product.poolHealth} />
                  <div className='text-xs text-muted-foreground'>{product.channelPool.length} bound channels</div>
                </div>
              </TableCell>
              <TableCell className='max-w-[280px] whitespace-normal text-xs'>{product.allowedModels.join(', ')}</TableCell>
              <TableCell>
                {product.activeKeyCount}/{product.keyCount} active
              </TableCell>
              <TableCell className='text-right'>{formatCurrency(product.monthlyCost)}</TableCell>
              <TableCell className='text-right'>
                <Button variant='ghost' size='sm' asChild>
                  <a href={`/relay-subkeys/products/${product.id}`}>Open</a>
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableFrame>
  );
}

function KeyTable({ keys }: { keys: RelayKey[] }) {
  if (keys.length === 0) {
    return <EmptyState title='No relay sub-keys' description='Issue a sub-key to make a shared product usable by a project.' />;
  }

  return (
    <TableFrame title='Sub-key inventory' description='Persistent key states are shown alongside runtime-derived risk badges.'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Key</TableHead>
            <TableHead>Project</TableHead>
            <TableHead>Product</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Derived state</TableHead>
            <TableHead className='text-right'>Today</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {keys.map((key) => (
            <TableRow key={key.id}>
              <TableCell className='whitespace-normal'>
                <div className='font-medium'>{key.name}</div>
                <div className='text-xs text-muted-foreground'>{key.maskedKey}</div>
              </TableCell>
              <TableCell>{key.projectName}</TableCell>
              <TableCell>{key.productName}</TableCell>
              <TableCell>
                <KeyStatusBadge status={key.status} />
              </TableCell>
              <TableCell className='whitespace-normal'>
                <DerivedStateBadges states={key.derivedStates} />
              </TableCell>
              <TableCell className='text-right'>
                <div>{formatNumber(key.usage.todayRequests)} req</div>
                <div className='text-xs text-muted-foreground'>{formatNumber(key.usage.todayTokens)} tokens</div>
              </TableCell>
              <TableCell className='text-right'>
                <Button variant='ghost' size='sm' asChild>
                  <a href={`/relay-subkeys/keys/${key.id}`}>Open</a>
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableFrame>
  );
}

function RequestTraceTable({ requests }: { requests: RelayRequestTrace[] }) {
  if (requests.length === 0) {
    return <EmptyState title='No request traces' description='Relay request facts will appear here after compatible API traffic is processed.' />;
  }

  return (
    <TableFrame title='Request and settlement trace' description='Request success and charge success are separate states for relay troubleshooting.'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Project / Key</TableHead>
            <TableHead>Product / Model</TableHead>
            <TableHead>Channel</TableHead>
            <TableHead>Failure stage</TableHead>
            <TableHead>Settlement</TableHead>
            <TableHead className='text-right'>Cost</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {requests.map((request) => (
            <TableRow key={request.id}>
              <TableCell>{formatDateTime(request.createdAt)}</TableCell>
              <TableCell className='whitespace-normal'>
                <div className='font-medium'>{request.projectName}</div>
                <div className='text-xs text-muted-foreground'>{request.keyName}</div>
              </TableCell>
              <TableCell className='whitespace-normal'>
                <div>{request.productName}</div>
                <div className='text-xs text-muted-foreground'>{request.modelId}</div>
              </TableCell>
              <TableCell className='whitespace-normal'>{request.channelName ?? '-'}</TableCell>
              <TableCell>
                <div className='space-y-1'>
                  <StatusBadge variant={failureStageVariant(request.failureStage)}>{request.failureStage}</StatusBadge>
                  {request.errorMessage ? <div className='max-w-[280px] whitespace-normal text-xs text-muted-foreground'>{request.errorMessage}</div> : null}
                </div>
              </TableCell>
              <TableCell>
                <div className='space-y-1'>
                  <Badge variant={request.charged ? 'default' : 'outline'}>{request.charged ? 'charged' : 'not charged'}</Badge>
                  <div className='text-xs text-muted-foreground'>{request.settlementStatus}</div>
                </div>
              </TableCell>
              <TableCell className='text-right'>{formatCurrency(request.chargeAmount)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableFrame>
  );
}

export function RelaySubkeysLayout() {
  const tt = useRelayText();
  return (
    <>
      <Header fixed>
        <div className='flex flex-1 flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div>
            <h2 className='text-xl font-bold tracking-tight'>{tt('relaySubkeys.operator.title', 'Relay Sub-Key Operations')}</h2>
            <p className='text-sm text-muted-foreground'>
              {tt('relaySubkeys.operator.description', 'Manage products, channel pools, sub-keys, wallets, and request traces for shared relay capacity.')}
            </p>
          </div>
          <Badge variant='secondary'>{tt('relaySubkeys.operator.badge', 'Frontend MVP')}</Badge>
        </div>
      </Header>
      <Main fixed className='space-y-6 overflow-y-auto'>
        <OperatorNav />
        <Outlet />
      </Main>
    </>
  );
}

export function RelaySubkeysOverviewPage() {
  const tt = useRelayText();
  const productsQuery = useRelayProductsQuery();
  const keysQuery = useRelayKeysQuery();
  const requestsQuery = useRelayRequestTraceQuery();
  const healthQuery = useRelayChannelPoolHealthQuery();

  if (productsQuery.isLoading || keysQuery.isLoading || requestsQuery.isLoading || healthQuery.isLoading) return <LoadingCards />;
  if (productsQuery.error || keysQuery.error || requestsQuery.error || healthQuery.error) {
    return <ErrorState error={productsQuery.error ?? keysQuery.error ?? requestsQuery.error ?? healthQuery.error} />;
  }

  const products = productsQuery.data ?? [];
  const keys = keysQuery.data ?? [];
  const requests = requestsQuery.data ?? [];
  const pools = healthQuery.data ?? [];
  const activeProducts = products.filter((product) => product.status === 'active').length;
  const degradedPools = pools.filter((pool) => pool.status !== 'healthy').length;
  const failedRequests = requests.filter((request) => request.status === 'failed').length;
  const activeKeys = keys.filter((key) => key.status === 'active').length;

  return (
    <div className='space-y-6'>
      <PageHeader
        title={tt('relaySubkeys.overview.title', 'Operator overview')}
        description={tt('relaySubkeys.overview.description', 'The MVP surface keeps the manual relay lifecycle visible from product setup to settlement troubleshooting.')}
        actions={
          <>
            <Button asChild>
              <a href='/relay-subkeys/products/create'>Create product</a>
            </Button>
            <Button variant='outline' asChild>
              <a href='/relay-subkeys/keys/create'>Issue sub-key</a>
            </Button>
          </>
        }
      />
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard label='Active products' value={`${activeProducts}/${products.length}`} hint='Products can issue keys only when the channel pool is ready.' />
        <MetricCard label='Active sub-keys' value={`${activeKeys}/${keys.length}`} hint='Derived badges call out low balance, expiry, quota, and pool risk.' />
        <MetricCard label='Degraded pools' value={formatNumber(degradedPools)} hint='Pool health separates upstream capacity issues from customer key issues.' />
        <MetricCard label='Failed traces' value={formatNumber(failedRequests)} hint='Request facts include failure stage and settlement outcome.' />
      </div>
      <div className='grid gap-6 xl:grid-cols-2'>
        <ProductTable products={products.slice(0, 3)} />
        <KeyTable keys={keys.slice(0, 3)} />
      </div>
    </div>
  );
}

export function RelayProductListPage() {
  const productsQuery = useRelayProductsQuery();
  const [statusFilter, setStatusFilter] = useState<'all' | RelayProductStatus>('all');
  const [search, setSearch] = useState('');

  const products = useMemo(() => {
    const rows = productsQuery.data ?? [];
    return rows.filter((product) => {
      const matchesStatus = statusFilter === 'all' || product.status === statusFilter;
      const query = search.toLowerCase();
      const matchesSearch = !query || product.name.toLowerCase().includes(query) || product.code.toLowerCase().includes(query);
      return matchesStatus && matchesSearch;
    });
  }, [productsQuery.data, search, statusFilter]);

  if (productsQuery.isLoading) return <LoadingCards />;
  if (productsQuery.error) return <ErrorState error={productsQuery.error} />;

  return (
    <div className='space-y-6'>
      <PageHeader
        title='Shared-capacity products'
        description='Create and review sellable relay products before issuing downstream sub-keys.'
        actions={
          <Button asChild>
            <a href='/relay-subkeys/products/create'>Create product</a>
          </Button>
        }
      />
      <Card>
        <CardContent className='flex flex-col gap-3 pt-6 md:flex-row'>
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder='Search product code or name...' />
          <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as 'all' | RelayProductStatus)}>
            <SelectTrigger className='w-full md:w-[180px]'>
              <SelectValue placeholder='Status' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>All statuses</SelectItem>
              <SelectItem value='draft'>Draft</SelectItem>
              <SelectItem value='active'>Active</SelectItem>
              <SelectItem value='archived'>Archived</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>
      <ProductTable products={products} />
    </div>
  );
}

export function RelayProductCreatePage() {
  const mutation = useCreateRelayProductMutation();
  const [code, setCode] = useState('gpt-shared-starter');
  const [name, setName] = useState('GPT Shared Starter');
  const [providerType, setProviderType] = useState<RelayProviderType>('openai_compatible');
  const [createdProduct, setCreatedProduct] = useState<RelayProduct | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate(
      {
        code,
        name,
        providerType,
        billingMode: 'prepaid',
        allowedModels: ['gpt-4.1-mini', 'gpt-4o-mini'],
        defaultTimeoutMs: 60000,
        description: 'Operator-created frontend MVP product draft.',
      },
      { onSuccess: setCreatedProduct }
    );
  };

  return (
    <div className='space-y-6'>
      <PageHeader title='Create relay product' description='Create the product shell first; bind channels from the product detail page before activation.' />
      <Card>
        <CardHeader>
          <CardTitle>Product metadata</CardTitle>
          <CardDescription>Mock fallback keeps the UI usable when REST mode is not enabled.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className='grid gap-4 md:grid-cols-2' onSubmit={handleSubmit}>
            <Input value={code} onChange={(event) => setCode(event.target.value)} placeholder='Product code' />
            <Input value={name} onChange={(event) => setName(event.target.value)} placeholder='Display name' />
            <Select value={providerType} onValueChange={(value) => setProviderType(value as RelayProviderType)}>
              <SelectTrigger>
                <SelectValue placeholder='Provider type' />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='claudecode'>Claude Code</SelectItem>
                <SelectItem value='codex'>Codex</SelectItem>
                <SelectItem value='openai_compatible'>OpenAI compatible</SelectItem>
              </SelectContent>
            </Select>
            <Button type='submit' disabled={mutation.isPending}>
              {mutation.isPending ? 'Saving draft...' : 'Save draft'}
            </Button>
          </form>
        </CardContent>
      </Card>
      {createdProduct ? (
        <Alert>
          <AlertTitle>Draft product created in UI state</AlertTitle>
          <AlertDescription>
            {createdProduct.name} is ready for channel-pool binding. Backend persistence will be used when VITE_RELAY_SUBKEYS_API_MODE=rest.
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

export function RelayProductDetailPage({ productId }: DetailPageProps) {
  const productQuery = useRelayProductDetailQuery(productId);
  const channelQuery = useRelayChannelPoolQuery(productId);
  const keysQuery = useRelayKeysQuery();
  const updateMutation = useUpdateRelayProductMutation();
  const bindMutation = useBindRelayChannelMutation();
  const updateBindingMutation = useUpdateRelayChannelBindingMutation();
  const deleteBindingMutation = useDeleteRelayChannelBindingMutation();
  const { canAccessScopes } = useRoutePermissions();
  const canWriteChannels = canAccessScopes(['write_channels'], 'system');

  if (productQuery.isLoading || channelQuery.isLoading || keysQuery.isLoading) return <LoadingCards />;
  if (productQuery.error || channelQuery.error || keysQuery.error) return <ErrorState error={productQuery.error ?? channelQuery.error ?? keysQuery.error} />;

  const product = productQuery.data;
  if (!product) return <EmptyState title='Product not found' description='The requested relay product does not exist in the current dataset.' />;

  const channels = channelQuery.data ?? [];
  const assignedKeys = (keysQuery.data ?? []).filter((key) => key.productId === product.id);

  return (
    <div className='space-y-6'>
      <PageHeader
        title={product.name}
        description={`${product.code} uses ${product.billingMode} billing and ${product.defaultTimeoutMs / 1000}s timeout.`}
        actions={
          canWriteChannels ? (
            <>
              <Button variant='outline' onClick={() => updateMutation.mutate({ id: product.id, input: { status: product.status === 'active' ? 'draft' : 'active' } })}>
                {product.status === 'active' ? 'Return to draft' : 'Activate product'}
              </Button>
              <Button
                onClick={() =>
                  bindMutation.mutate({
                    productId: product.id,
                    channelId: 'preview-channel',
                    priority: channels.length + 1,
                    weight: 10,
                    modelFilter: product.allowedModels.slice(0, 1),
                    allowFallback: true,
                  })
                }
              >
                Bind preview channel
              </Button>
            </>
          ) : undefined
        }
      />
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard label='Status' value={product.status} hint='Only active products should receive new traffic.' />
        <MetricCard label='Pool health' value={product.poolHealth} hint='Healthy candidates are required before activation.' />
        <MetricCard label='Issued keys' value={`${product.activeKeyCount}/${product.keyCount}`} hint='Active keys currently mapped to this product.' />
        <MetricCard label='Monthly tokens' value={formatNumber(product.monthlyTokenCount)} hint={formatCurrency(product.monthlyCost)} />
      </div>
      <Tabs defaultValue='pool'>
        <TabsList>
          <TabsTrigger value='pool'>Channel pool</TabsTrigger>
          <TabsTrigger value='models'>Models</TabsTrigger>
          <TabsTrigger value='keys'>Assigned keys</TabsTrigger>
        </TabsList>
        <TabsContent value='pool' className='space-y-4'>
          <TableFrame title='Bound upstream channels' description='Priority, weight, fallback behavior, and health explain product capacity.'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Channel</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Weight</TableHead>
                  <TableHead>Health</TableHead>
                  <TableHead>Quota</TableHead>
                  <TableHead>Models</TableHead>
                  {canWriteChannels ? <TableHead className='text-right'>Actions</TableHead> : null}
                </TableRow>
              </TableHeader>
              <TableBody>
                {channels.map((channel) => (
                  <TableRow key={channel.id}>
                    <TableCell className='whitespace-normal'>
                      <div className='font-medium'>{channel.channelName}</div>
                      <div className='text-xs text-muted-foreground'>{channel.unavailableReason ?? `${channel.latencyMs} ms latency`}</div>
                    </TableCell>
                    <TableCell>{channel.priority}</TableCell>
                    <TableCell>{channel.weight}</TableCell>
                    <TableCell>
                      <HealthBadge health={channel.health} />
                    </TableCell>
                    <TableCell className='min-w-[160px]'>
                      <Progress value={channel.quotaRemainingPercent} />
                      <div className='mt-1 text-xs text-muted-foreground'>{channel.quotaRemainingPercent}% remaining</div>
                    </TableCell>
                    <TableCell className='whitespace-normal text-xs'>{channel.modelFilter.join(', ')}</TableCell>
                    {canWriteChannels ? (
                      <TableCell className='space-x-2 text-right'>
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={updateBindingMutation.isPending}
                          onClick={() =>
                            updateBindingMutation.mutate({
                              id: channel.id,
                              productId: product.id,
                              status: channel.status === 'active' ? 'paused' : 'active',
                            })
                          }
                        >
                          {channel.status === 'active' ? 'Pause' : 'Resume'}
                        </Button>
                        <Button
                          variant='destructive'
                          size='sm'
                          disabled={deleteBindingMutation.isPending}
                          onClick={() => deleteBindingMutation.mutate({ id: channel.id, productId: product.id })}
                        >
                          Remove
                        </Button>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableFrame>
        </TabsContent>
        <TabsContent value='models'>
          <Card>
            <CardHeader>
              <CardTitle>Allowed models</CardTitle>
              <CardDescription>Model scope must remain consistent with the bound channel filters.</CardDescription>
            </CardHeader>
            <CardContent className='flex flex-wrap gap-2'>
              {product.allowedModels.map((model) => (
                <Badge key={model} variant='outline'>
                  {model}
                </Badge>
              ))}
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value='keys'>
          <KeyTable keys={assignedKeys} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export function RelayKeyListPage() {
  const keysQuery = useRelayKeysQuery();
  const [statusFilter, setStatusFilter] = useState<'all' | RelayKeyStatus>('all');
  const [search, setSearch] = useState('');

  const keys = useMemo(() => {
    const rows = keysQuery.data ?? [];
    return rows.filter((key) => {
      const matchesStatus = statusFilter === 'all' || key.status === statusFilter;
      const query = search.toLowerCase();
      const matchesSearch = !query || key.name.toLowerCase().includes(query) || key.projectName.toLowerCase().includes(query) || key.maskedKey.toLowerCase().includes(query);
      return matchesStatus && matchesSearch;
    });
  }, [keysQuery.data, search, statusFilter]);

  if (keysQuery.isLoading) return <LoadingCards />;
  if (keysQuery.error) return <ErrorState error={keysQuery.error} />;

  return (
    <div className='space-y-6'>
      <PageHeader
        title='Relay sub-key inventory'
        description='Search issued downstream keys and distinguish persisted status from derived runtime risk.'
        actions={
          <Button asChild>
            <a href='/relay-subkeys/keys/create'>Issue sub-key</a>
          </Button>
        }
      />
      <Card>
        <CardContent className='flex flex-col gap-3 pt-6 md:flex-row'>
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder='Search key, project, or masked secret...' />
          <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as 'all' | RelayKeyStatus)}>
            <SelectTrigger className='w-full md:w-[180px]'>
              <SelectValue placeholder='Status' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>All statuses</SelectItem>
              <SelectItem value='active'>Active</SelectItem>
              <SelectItem value='suspended'>Suspended</SelectItem>
              <SelectItem value='exhausted'>Exhausted</SelectItem>
              <SelectItem value='archived'>Archived</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>
      <KeyTable keys={keys} />
    </div>
  );
}

export function RelayKeyCreatePage() {
  const productsQuery = useRelayProductsQuery();
  const mutation = useCreateRelayKeyMutation();
  const [productId, setProductId] = useState('prod-gpt-shared');
  const [projectId, setProjectId] = useState('project-alpha');
  const [name, setName] = useState('New project relay key');
  const [createdKey, setCreatedKey] = useState<RelayKey | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setCreatedKey(null);
    mutation.mutate(
      {
        projectId,
        productId,
        name,
        balanceMode: 'prepaid',
        initialBalance: 100,
        limits: { dailyRequestLimit: 10000, dailyTokenLimit: 5000000, monthlyCostLimit: 500, concurrencyLimit: 8 },
      },
      { onSuccess: setCreatedKey }
    );
  };

  const oneTimeCredential = createdKey?.plaintextKey;

  return (
    <div className='space-y-6'>
      <PageHeader title='Issue relay sub-key' description='Bind a project to an active product and prepare the one-time plaintext credential handoff.' />
      <Card>
        <CardHeader>
          <CardTitle>Issuance form</CardTitle>
          <CardDescription>Creation uses mock fallback when REST mode is not enabled.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className='grid gap-4 md:grid-cols-2' onSubmit={handleSubmit}>
            <Input value={projectId} onChange={(event) => setProjectId(event.target.value)} placeholder='Project ID' />
            <Input value={name} onChange={(event) => setName(event.target.value)} placeholder='Display name' />
            <Select value={productId} onValueChange={setProductId}>
              <SelectTrigger>
                <SelectValue placeholder='Relay product' />
              </SelectTrigger>
              <SelectContent>
                {(productsQuery.data ?? []).map((product) => (
                  <SelectItem key={product.id} value={product.id}>
                    {product.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button type='submit' disabled={mutation.isPending || productsQuery.isLoading}>
              {mutation.isPending ? 'Issuing...' : 'Issue key'}
            </Button>
          </form>
        </CardContent>
      </Card>
      {createdKey ? (
        <Alert>
          <AlertTitle>{oneTimeCredential ? 'One-time plaintext credential' : 'Relay sub-key created'}</AlertTitle>
          <AlertDescription className='space-y-2'>
            {oneTimeCredential ? (
              <>
                <p>Copy and store this value now. It will not be shown again in lists or detail pages.</p>
                <div className='rounded-lg border bg-muted/40 p-3 font-mono text-sm text-foreground'>{oneTimeCredential}</div>
              </>
            ) : (
              <p>Plaintext was not returned by this response. Use the masked identifier {createdKey.maskedKey} for tracking and re-issue if the secret was not captured.</p>
            )}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

export function RelayKeyDetailPage({ keyId }: DetailPageProps) {
  const keyQuery = useRelayKeyDetailQuery(keyId);
  const walletQuery = useRelayWalletQuery(keyId);
  const ledgerQuery = useRelayLedgerEntriesQuery(keyId);
  const requestsQuery = useRelayRequestTraceQuery();
  const suspendMutation = useSuspendRelayKeyMutation();
  const resumeMutation = useResumeRelayKeyMutation();
  const archiveMutation = useArchiveRelayKeyMutation();
  const limitMutation = useAdjustRelayKeyLimitMutation();
  const { canAccessScopes } = useRoutePermissions();
  const canWriteApiKeys = canAccessScopes(['write_api_keys'], 'system');

  if (keyQuery.isLoading || walletQuery.isLoading || ledgerQuery.isLoading || requestsQuery.isLoading) return <LoadingCards />;
  if (keyQuery.error || walletQuery.error || ledgerQuery.error || requestsQuery.error) {
    return <ErrorState error={keyQuery.error ?? walletQuery.error ?? ledgerQuery.error ?? requestsQuery.error} />;
  }

  const key = keyQuery.data;
  if (!key) return <EmptyState title='Sub-key not found' description='The requested relay sub-key does not exist in the current dataset.' />;

  const wallet = walletQuery.data;
  const ledger = ledgerQuery.data ?? [];
  const requests = (requestsQuery.data ?? []).filter((request) => request.keyName === key.name);

  return (
    <div className='space-y-6'>
      <PageHeader
        title={key.name}
        description={`${key.projectName} uses ${key.productName}. Last used ${formatDateTime(key.lastUsedAt)}.`}
        actions={
          <>
            <Button variant='outline' asChild>
              <a href={`/relay-subkeys/keys/${key.id}/billing`}>Open wallet</a>
            </Button>
            {canWriteApiKeys ? (
              <>
                <Button variant='secondary' onClick={() => suspendMutation.mutate({ id: key.id, note: 'Operator pause from MVP UI' })} disabled={key.status === 'suspended'}>
                  Suspend
                </Button>
                <Button variant='outline' onClick={() => resumeMutation.mutate({ id: key.id })} disabled={key.status === 'active'}>
                  Resume
                </Button>
                <Button variant='destructive' onClick={() => archiveMutation.mutate({ id: key.id, note: 'Archive from MVP UI' })} disabled={key.status === 'archived'}>
                  Archive
                </Button>
              </>
            ) : null}
          </>
        }
      />
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard label='Status' value={key.status} hint='Persisted state controls runtime access.' />
        <MetricCard label='Available balance' value={wallet ? formatCurrency(wallet.availableAmount, wallet.currency) : '-'} hint='Low-balance badges are derived from wallet thresholds.' />
        <MetricCard label='Today requests' value={formatNumber(key.usage.todayRequests)} hint={`${formatNumber(key.usage.todayTokens)} tokens today`} />
        <MetricCard label='Monthly cost' value={formatCurrency(key.usage.monthlyCost)} hint={`Expires ${formatDateTime(key.expiresAt)}`} />
      </div>
      <Tabs defaultValue='overview'>
        <TabsList>
          <TabsTrigger value='overview'>Overview</TabsTrigger>
          <TabsTrigger value='wallet'>Wallet & ledger</TabsTrigger>
          <TabsTrigger value='requests'>Recent requests</TabsTrigger>
          <TabsTrigger value='limits'>Limits</TabsTrigger>
        </TabsList>
        <TabsContent value='overview' className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
          <Card>
            <CardHeader>
              <CardTitle>Credential and risk</CardTitle>
              <CardDescription>Plaintext is never replayed after creation; operational state explains runtime decisions.</CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='rounded-lg border p-4 font-mono text-sm'>{key.maskedKey}</div>
              <div className='flex flex-wrap gap-2'>
                <KeyStatusBadge status={key.status} />
                <DerivedStateBadges states={key.derivedStates} />
              </div>
              {key.usage.recentFailure ? <Alert><AlertTitle>Recent failure</AlertTitle><AlertDescription>{key.usage.recentFailure}</AlertDescription></Alert> : null}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Runtime identifiers</CardTitle>
              <CardDescription>Used by support when correlating requests and ledger entries.</CardDescription>
            </CardHeader>
            <CardContent className='space-y-2 text-sm'>
              <div>API key ID: {key.apiKeyId}</div>
              <div>Project ID: {key.projectId}</div>
              <div>Product ID: {key.productId}</div>
              <div>Base URL: {key.baseUrl}</div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value='wallet'>
          <RelayWalletAndLedger keyId={key.id} wallet={wallet} ledger={ledger} />
        </TabsContent>
        <TabsContent value='requests'>
          <RequestTraceTable requests={requests} />
        </TabsContent>
        <TabsContent value='limits'>
          <Card>
            <CardHeader>
              <CardTitle>Usage guards</CardTitle>
              <CardDescription>Daily limits are enforced; monthly cost is a soft preflight guard and concurrency is an MVP preview.</CardDescription>
            </CardHeader>
            <CardContent className='grid gap-4 md:grid-cols-4'>
              <MetricCard label='Daily requests' value={formatNumber(key.limits.dailyRequestLimit)} hint='Hard cap before quota_reached.' />
              <MetricCard label='Daily tokens' value={formatNumber(key.limits.dailyTokenLimit)} hint='Token guard for shared pool use.' />
              <MetricCard label='Monthly cost' value={formatCurrency(key.limits.monthlyCostLimit)} hint='Soft preflight guard; not a settlement hard cap.' />
              <MetricCard label='Concurrency' value={formatNumber(key.limits.concurrencyLimit)} hint='Preview value; positive limits are not enforced in MVP.' />
              {canWriteApiKeys ? (
                <div className='md:col-span-4'>
                  <Button
                    variant='outline'
                    onClick={() =>
                      limitMutation.mutate({
                        id: key.id,
                        limits: { ...key.limits, concurrencyLimit: key.limits.concurrencyLimit + 1 },
                      })
                    }
                  >
                    Increase preview concurrency
                  </Button>
                </div>
              ) : null}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function RelayWalletAndLedger({
  keyId,
  wallet,
  ledger,
}: {
  keyId: string;
  wallet: ReturnType<typeof useRelayWalletQuery>['data'];
  ledger: ReturnType<typeof useRelayLedgerEntriesQuery>['data'];
}) {
  const rechargeMutation = useRechargeRelayWalletMutation();
  const { canAccessScopes } = useRoutePermissions();
  const canWriteApiKeys = canAccessScopes(['write_api_keys'], 'system');

  return (
    <div className='space-y-6'>
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard label='Available' value={wallet ? formatCurrency(wallet.availableAmount, wallet.currency) : '-'} hint='Amount available for new relay requests.' />
        <MetricCard label='Frozen' value={wallet ? formatCurrency(wallet.frozenAmount, wallet.currency) : '-'} hint='Reserved for in-flight settlement.' />
        <MetricCard label='Recharged' value={wallet ? formatCurrency(wallet.totalRecharged, wallet.currency) : '-'} hint='Manual top-ups and refunds.' />
        <MetricCard label='Spent' value={wallet ? formatCurrency(wallet.totalSpent, wallet.currency) : '-'} hint='Usage settlement ledger total.' />
      </div>
      {canWriteApiKeys ? (
        <Card>
          <CardHeader>
            <CardTitle>Operator adjustment</CardTitle>
            <CardDescription>Recharge uses mutation glue with mock fallback until backend endpoints land.</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => rechargeMutation.mutate({ relayKeyId: keyId, amount: 100, note: 'MVP preview recharge' })} disabled={rechargeMutation.isPending}>
              {rechargeMutation.isPending ? 'Posting...' : 'Post $100 preview recharge'}
            </Button>
          </CardContent>
        </Card>
      ) : null}
      <TableFrame title='Ledger entries' description='Recharge, usage settlement, refunds, and adjustments must remain explainable.'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Note</TableHead>
              <TableHead>Reference</TableHead>
              <TableHead className='text-right'>Amount</TableHead>
              <TableHead className='text-right'>Balance after</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(ledger ?? []).map((entry) => (
              <TableRow key={entry.id}>
                <TableCell>{formatDateTime(entry.createdAt)}</TableCell>
                <TableCell>
                  <StatusBadge variant={ledgerTypeVariant(entry.type)}>{entry.type}</StatusBadge>
                </TableCell>
                <TableCell className='whitespace-normal'>{entry.note}</TableCell>
                <TableCell>{entry.referenceId ?? '-'}</TableCell>
                <TableCell className='text-right'>{formatCurrency(entry.amount, entry.currency)}</TableCell>
                <TableCell className='text-right'>{formatCurrency(entry.balanceAfter, entry.currency)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableFrame>
    </div>
  );
}

export function RelayKeyBillingPage({ keyId }: DetailPageProps) {
  const keyQuery = useRelayKeyDetailQuery(keyId);
  const walletQuery = useRelayWalletQuery(keyId);
  const ledgerQuery = useRelayLedgerEntriesQuery(keyId);

  if (keyQuery.isLoading || walletQuery.isLoading || ledgerQuery.isLoading) return <LoadingCards />;
  if (keyQuery.error || walletQuery.error || ledgerQuery.error) return <ErrorState error={keyQuery.error ?? walletQuery.error ?? ledgerQuery.error} />;
  if (!keyQuery.data) return <EmptyState title='Sub-key not found' description='Wallet and ledger data could not be resolved.' />;

  return (
    <div className='space-y-6'>
      <PageHeader title={`${keyQuery.data.name} wallet`} description='Recharge, refund, manual adjustment, and usage settlement evidence for support workflows.' />
      <RelayWalletAndLedger keyId={keyQuery.data.id} wallet={walletQuery.data} ledger={ledgerQuery.data} />
    </div>
  );
}

export function RelayRequestListPage() {
  const requestsQuery = useRelayRequestTraceQuery();
  const [stage, setStage] = useState<'all' | RelayFailureStage>('all');
  const [search, setSearch] = useState('');

  const requests = useMemo(() => {
    const rows = requestsQuery.data ?? [];
    return rows.filter((request) => {
      const matchesStage = stage === 'all' || request.failureStage === stage;
      const query = search.toLowerCase();
      const matchesSearch =
        !query ||
        request.projectName.toLowerCase().includes(query) ||
        request.keyName.toLowerCase().includes(query) ||
        request.productName.toLowerCase().includes(query) ||
        request.modelId.toLowerCase().includes(query);
      return matchesStage && matchesSearch;
    });
  }, [requestsQuery.data, search, stage]);

  if (requestsQuery.isLoading) return <LoadingCards />;
  if (requestsQuery.error) return <ErrorState error={requestsQuery.error} />;

  return (
    <div className='space-y-6'>
      <PageHeader title='Relay request troubleshooting' description='Filter by key, product, channel, or failure stage to explain where a request failed.' />
      <Card>
        <CardContent className='flex flex-col gap-3 pt-6 md:flex-row'>
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder='Search project, key, product, or model...' />
          <Select value={stage} onValueChange={(value) => setStage(value as 'all' | RelayFailureStage)}>
            <SelectTrigger className='w-full md:w-[220px]'>
              <SelectValue placeholder='Failure stage' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>All stages</SelectItem>
              <SelectItem value='none'>None</SelectItem>
              <SelectItem value='auth'>Auth</SelectItem>
              <SelectItem value='key_validation'>Key validation</SelectItem>
              <SelectItem value='routing'>Routing</SelectItem>
              <SelectItem value='upstream'>Upstream</SelectItem>
              <SelectItem value='settlement'>Settlement</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>
      <RequestTraceTable requests={requests} />
    </div>
  );
}

export function RelayChannelPoolHealthPage() {
  const healthQuery = useRelayChannelPoolHealthQuery();

  if (healthQuery.isLoading) return <LoadingCards />;
  if (healthQuery.error) return <ErrorState error={healthQuery.error} />;

  const pools = healthQuery.data ?? [];
  const atRisk = pools.filter((pool) => pool.status !== 'healthy').length;

  return (
    <div className='space-y-6'>
      <PageHeader title='Channel pool health' description='Inspect shared upstream risk by product before blaming customer balances or keys.' />
      <div className='grid gap-4 md:grid-cols-3'>
        <MetricCard label='Products watched' value={formatNumber(pools.length)} hint='Every relay product with a configured or missing pool.' />
        <MetricCard label='At-risk pools' value={formatNumber(atRisk)} hint='Degraded or unavailable pools need operator action.' />
        <MetricCard label='Healthy pools' value={formatNumber(pools.length - atRisk)} hint='Ready for key issuance and traffic.' />
      </div>
      <div className='space-y-6'>
        {pools.map((pool) => (
          <Card key={pool.productId}>
            <CardHeader>
              <div className='flex flex-wrap items-center justify-between gap-3'>
                <div>
                  <CardTitle>{pool.productName}</CardTitle>
                  <CardDescription>{pool.riskReason ?? 'No active shared-pool risk reported.'}</CardDescription>
                </div>
                <HealthBadge health={pool.status} />
              </div>
            </CardHeader>
            <CardContent>
              <div className='grid gap-4 md:grid-cols-3'>
                <MetricCard label='Healthy' value={formatNumber(pool.healthyChannels)} hint='Can receive traffic.' />
                <MetricCard label='Degraded' value={formatNumber(pool.degradedChannels)} hint='Use with fallback caution.' />
                <MetricCard label='Unavailable' value={formatNumber(pool.unavailableChannels)} hint='Excluded from routing.' />
              </div>
              {pool.channels.length > 0 ? (
                <div className='mt-6 overflow-x-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Channel</TableHead>
                        <TableHead>Health</TableHead>
                        <TableHead>Fallback</TableHead>
                        <TableHead>Quota</TableHead>
                        <TableHead>Error rate</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {pool.channels.map((channel) => (
                        <TableRow key={channel.id}>
                          <TableCell className='whitespace-normal'>{channel.channelName}</TableCell>
                          <TableCell>
                            <HealthBadge health={channel.health} />
                          </TableCell>
                          <TableCell>{channel.allowFallback ? 'allowed' : 'blocked'}</TableCell>
                          <TableCell className='min-w-[160px]'>
                            <Progress value={channel.quotaRemainingPercent} />
                            <div className='mt-1 text-xs text-muted-foreground'>{channel.quotaRemainingPercent}% remaining</div>
                          </TableCell>
                          <TableCell>{channel.errorRatePercent}%</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : (
                <EmptyState title='No channels bound' description='This product cannot be activated until at least one upstream channel is bound.' />
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
