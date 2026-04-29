import { type ReactNode, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Header } from '@/components/layout/header';
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
import { Textarea } from '@/components/ui/textarea';
import {
  type RelayDerivedState,
  type RelayFailureStage,
  type RelayHealthStatus,
  type RelayKey,
  type RelayKeyStatus,
  type RelayProduct,
  type RelayProductStatus,
  type RelayRequestTrace,
  type RelayWalletLedgerType,
  useProjectRelayKeysQuery,
  useProjectRelayOverviewQuery,
  useProjectRelayProductsQuery,
  useProjectRelayUsageQuery,
  useRelayKeyDetailQuery,
  useRelayRequestTraceQuery,
  useRelayWalletQuery,
} from '../relay-subkeys/data';

interface ProjectKeyDetailProps {
  keyId?: string;
}

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const projectNav = [
  { labelKey: 'relaySubkeys.project.nav.overview', fallback: 'Overview', href: '/project/relay-subkeys' },
  { labelKey: 'relaySubkeys.project.nav.products', fallback: 'Products', href: '/project/relay-subkeys/products' },
  { labelKey: 'relaySubkeys.project.nav.keys', fallback: 'Keys', href: '/project/relay-subkeys/keys' },
  { labelKey: 'relaySubkeys.project.nav.usage', fallback: 'Usage', href: '/project/relay-subkeys/usage' },
  { labelKey: 'relaySubkeys.project.nav.getStarted', fallback: 'Get started', href: '/project/relay-subkeys/get-started' },
  { labelKey: 'relaySubkeys.project.nav.verify', fallback: 'Verify', href: '/project/relay-subkeys/verify' },
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

function keyStatusVariant(status: RelayKeyStatus): BadgeVariant {
  if (status === 'active') return 'default';
  if (status === 'exhausted') return 'destructive';
  if (status === 'suspended') return 'secondary';
  return 'outline';
}

function productStatusVariant(status: RelayProductStatus): BadgeVariant {
  if (status === 'active') return 'default';
  if (status === 'archived') return 'outline';
  return 'secondary';
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

function KeyStatusBadge({ status }: { status: RelayKeyStatus }) {
  return <StatusBadge variant={keyStatusVariant(status)}>{status}</StatusBadge>;
}

function ProductStatusBadge({ status }: { status: RelayProductStatus }) {
  return <StatusBadge variant={productStatusVariant(status)}>{status}</StatusBadge>;
}

function HealthBadge({ health }: { health: RelayHealthStatus }) {
  return <StatusBadge variant={healthVariant(health)}>{health}</StatusBadge>;
}

function DerivedStateBadges({ states }: { states: RelayDerivedState[] }) {
  if (states.length === 0) return <Badge variant='outline'>ready</Badge>;
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

function ProjectShell({ title, description, actions, children }: { title: string; description: string; actions?: ReactNode; children: ReactNode }) {
  const tt = useRelayText();
  return (
    <>
      <Header fixed>
        <div className='flex flex-1 flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div>
            <div className='mb-2 flex flex-wrap gap-2'>
              <Badge>{tt('relaySubkeys.project.badge', 'Relay Sub-Keys')}</Badge>
              <Badge variant='secondary'>{tt('relaySubkeys.project.scope', 'Project view')}</Badge>
            </div>
            <h2 className='text-xl font-bold tracking-tight'>{title}</h2>
            <p className='text-sm text-muted-foreground'>{description}</p>
          </div>
          {actions ? <div className='flex flex-wrap gap-2'>{actions}</div> : null}
        </div>
      </Header>
      <Main className='space-y-6'>
        <div className='flex flex-wrap gap-2'>
          {projectNav.map((item) => (
            <Button key={item.href} variant='outline' size='sm' asChild>
              <a href={item.href}>{tt(item.labelKey, item.fallback)}</a>
            </Button>
          ))}
        </div>
        {children}
      </Main>
    </>
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

function ProductCards({ products }: { products: RelayProduct[] }) {
  if (products.length === 0) return <EmptyState title='No products available' description='Ask an operator to grant access to a relay product for this project.' />;

  return (
    <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
      {products.map((product) => (
        <Card key={product.id}>
          <CardHeader>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <CardTitle>{product.name}</CardTitle>
              <ProductStatusBadge status={product.status} />
            </div>
            <CardDescription>{product.description}</CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <div className='flex flex-wrap gap-2'>
              <HealthBadge health={product.poolHealth} />
              <Badge variant='outline'>{product.providerType}</Badge>
              <Badge variant='outline'>{product.billingMode}</Badge>
            </div>
            <div>
              <p className='mb-2 text-sm font-medium'>Supported models</p>
              <div className='flex flex-wrap gap-1'>
                {product.allowedModels.map((model) => (
                  <Badge key={model} variant='secondary'>
                    {model}
                  </Badge>
                ))}
              </div>
            </div>
            <div className='grid gap-3 text-sm md:grid-cols-2'>
              <div className='rounded-lg bg-muted/40 p-3'>Timeout {product.defaultTimeoutMs / 1000}s</div>
              <div className='rounded-lg bg-muted/40 p-3'>{product.channelPool.length} channels</div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ProjectKeyTable({ keys }: { keys: RelayKey[] }) {
  if (keys.length === 0) return <EmptyState title='No relay keys issued' description='This project does not have any relay sub-keys yet.' />;

  return (
    <TableFrame title='Project relay keys' description='Project users can inspect masked credentials and setup state without operator lifecycle controls.'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Key</TableHead>
            <TableHead>Product</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Derived state</TableHead>
            <TableHead className='text-right'>Balance risk</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {keys.map((key) => {
            const costPercent = key.limits.monthlyCostLimit > 0 ? Math.min(100, (key.usage.monthlyCost / key.limits.monthlyCostLimit) * 100) : 0;
            return (
              <TableRow key={key.id}>
                <TableCell className='whitespace-normal'>
                  <div className='font-medium'>{key.name}</div>
                  <div className='text-xs text-muted-foreground'>{key.maskedKey}</div>
                </TableCell>
                <TableCell>{key.productName}</TableCell>
                <TableCell>
                  <KeyStatusBadge status={key.status} />
                </TableCell>
                <TableCell className='whitespace-normal'>
                  <DerivedStateBadges states={key.derivedStates} />
                </TableCell>
                <TableCell className='min-w-[180px] text-right'>
                  <Progress value={costPercent} />
                  <div className='mt-1 text-xs text-muted-foreground'>
                    {formatCurrency(key.usage.monthlyCost)} / {formatCurrency(key.limits.monthlyCostLimit)}
                  </div>
                </TableCell>
                <TableCell className='text-right'>
                  <Button variant='ghost' size='sm' asChild>
                    <a href={`/project/relay-subkeys/keys/${key.id}`}>Open</a>
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableFrame>
  );
}

function RequestTraceTable({ requests }: { requests: RelayRequestTrace[] }) {
  if (requests.length === 0) return <EmptyState title='No request traces' description='Relay request activity for this project will appear after traffic is processed.' />;

  return (
    <TableFrame title='Recent relay requests' description='Failure stage helps distinguish setup issues from quota, balance, routing, or settlement problems.'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Key</TableHead>
            <TableHead>Product / Model</TableHead>
            <TableHead>Stage</TableHead>
            <TableHead>Settlement</TableHead>
            <TableHead className='text-right'>Cost</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {requests.map((request) => (
            <TableRow key={request.id}>
              <TableCell>{formatDateTime(request.createdAt)}</TableCell>
              <TableCell className='whitespace-normal'>{request.keyName}</TableCell>
              <TableCell className='whitespace-normal'>
                <div>{request.productName}</div>
                <div className='text-xs text-muted-foreground'>{request.modelId}</div>
              </TableCell>
              <TableCell>
                <div className='space-y-1'>
                  <StatusBadge variant={failureStageVariant(request.failureStage)}>{request.failureStage}</StatusBadge>
                  {request.errorMessage ? <div className='max-w-[260px] whitespace-normal text-xs text-muted-foreground'>{request.errorMessage}</div> : null}
                </div>
              </TableCell>
              <TableCell>{request.settlementStatus}</TableCell>
              <TableCell className='text-right'>{formatCurrency(request.chargeAmount)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableFrame>
  );
}

export function ProjectRelaySubkeysOverviewPage() {
  const tt = useRelayText();
  const overviewQuery = useProjectRelayOverviewQuery();

  if (overviewQuery.isLoading) return <ProjectShell title='Project Relay Sub-Keys' description='Loading project-scoped relay workspace.'><LoadingCards /></ProjectShell>;
  if (overviewQuery.error) return <ProjectShell title='Project Relay Sub-Keys' description='Project-scoped relay workspace.'><ErrorState error={overviewQuery.error} /></ProjectShell>;

  const overview = overviewQuery.data;
  const keys = overview?.keys ?? [];
  const wallets = overview?.wallets ?? [];
  const requests = overview?.recentRequests ?? [];
  const totalBalance = wallets.reduce((sum, wallet) => sum + wallet.availableAmount, 0);
  const failedRequests = requests.filter((request) => request.status === 'failed').length;

  return (
    <ProjectShell
      title={tt('relaySubkeys.project.overview.title', 'Project Relay Sub-Keys')}
      description={tt('relaySubkeys.project.overview.description', 'Inspect usable relay products, issued keys, usage, and setup health for the selected project.')}
      actions={
        <>
          <Button asChild><a href='/project/relay-subkeys/products'>Browse products</a></Button>
          <Button variant='outline' asChild><a href='/project/relay-subkeys/get-started'>Get started</a></Button>
        </>
      }
    >
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
        <MetricCard label='Available products' value={formatNumber(overview?.products.length ?? 0)} hint='Catalog items that can be used by this project.' />
        <MetricCard label='Issued keys' value={formatNumber(keys.length)} hint='Keys are read-only in project view for the MVP.' />
        <MetricCard label='Available balance' value={formatCurrency(totalBalance)} hint='Aggregate wallet balance across visible keys.' />
        <MetricCard label='Recent failures' value={formatNumber(failedRequests)} hint='Review setup and runtime failures before contacting support.' />
      </div>
      <div className='grid gap-6 xl:grid-cols-2'>
        <ProjectKeyTable keys={keys.slice(0, 3)} />
        <RequestTraceTable requests={requests.slice(0, 3)} />
      </div>
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysProductsPage() {
  const productsQuery = useProjectRelayProductsQuery();
  const [providerFilter, setProviderFilter] = useState('all');
  const [search, setSearch] = useState('');

  const products = useMemo(() => {
    const rows = productsQuery.data ?? [];
    return rows.filter((product) => {
      const query = search.toLowerCase();
      const matchesSearch = !query || product.name.toLowerCase().includes(query) || product.code.toLowerCase().includes(query);
      const matchesProvider = providerFilter === 'all' || product.providerType === providerFilter;
      return matchesSearch && matchesProvider;
    });
  }, [productsQuery.data, providerFilter, search]);

  return (
    <ProjectShell title='Relay Products' description='Review shared-capacity bundles and supported models available to this project.'>
      <PageHeader title='Product catalog' description='Project view is read-only: contact an operator for product activation, pool repair, or issuing new keys.' />
      <Card>
        <CardContent className='flex flex-col gap-3 pt-6 md:flex-row'>
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder='Search product code or model family...' />
          <Select value={providerFilter} onValueChange={setProviderFilter}>
            <SelectTrigger className='w-full md:w-[200px]'>
              <SelectValue placeholder='Provider' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>All providers</SelectItem>
              <SelectItem value='claudecode'>Claude Code</SelectItem>
              <SelectItem value='codex'>Codex</SelectItem>
              <SelectItem value='openai_compatible'>OpenAI compatible</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>
      {productsQuery.isLoading ? <LoadingCards /> : null}
      {productsQuery.error ? <ErrorState error={productsQuery.error} /> : null}
      {!productsQuery.isLoading && !productsQuery.error ? <ProductCards products={products} /> : null}
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysKeysPage() {
  const keysQuery = useProjectRelayKeysQuery();
  const [statusFilter, setStatusFilter] = useState<'all' | RelayKeyStatus>('all');
  const [search, setSearch] = useState('');

  const keys = useMemo(() => {
    const rows = keysQuery.data ?? [];
    return rows.filter((key) => {
      const query = search.toLowerCase();
      const matchesSearch = !query || key.name.toLowerCase().includes(query) || key.maskedKey.toLowerCase().includes(query) || key.productName.toLowerCase().includes(query);
      const matchesStatus = statusFilter === 'all' || key.status === statusFilter;
      return matchesSearch && matchesStatus;
    });
  }, [keysQuery.data, search, statusFilter]);

  return (
    <ProjectShell title='Project Sub-Keys' description='Inspect issued downstream credentials, runtime state, and onboarding links.'>
      <PageHeader
        title='Issued relay keys'
        description='Plaintext is not replayed after creation; use masked identifiers and detail pages for safe troubleshooting.'
        actions={<Button asChild><a href='/project/relay-subkeys/get-started'>Integration guide</a></Button>}
      />
      <Card>
        <CardContent className='flex flex-col gap-3 pt-6 md:flex-row'>
          <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder='Search key, product, or masked value...' />
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
      {keysQuery.isLoading ? <LoadingCards /> : null}
      {keysQuery.error ? <ErrorState error={keysQuery.error} /> : null}
      {!keysQuery.isLoading && !keysQuery.error ? <ProjectKeyTable keys={keys} /> : null}
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysKeyDetailPage({ keyId }: ProjectKeyDetailProps) {
  const keysQuery = useProjectRelayKeysQuery();
  const detailQuery = useRelayKeyDetailQuery(keyId);
  const requestQuery = useRelayRequestTraceQuery();
  const walletQuery = useRelayWalletQuery(keyId);
  const key = detailQuery.data ?? keysQuery.data?.find((candidate) => candidate.id === keyId) ?? keysQuery.data?.[0];
  const requests = (requestQuery.data ?? []).filter((request) => request.keyName === key?.name);
  const wallet = walletQuery.data;

  return (
    <ProjectShell title='Sub-Key Detail' description='View safe credential metadata, setup values, and project-scoped troubleshooting evidence.'>
      {(keysQuery.isLoading || detailQuery.isLoading || requestQuery.isLoading || walletQuery.isLoading) && !key ? <LoadingCards /> : null}
      {keysQuery.error || detailQuery.error || requestQuery.error || walletQuery.error ? <ErrorState error={keysQuery.error ?? detailQuery.error ?? requestQuery.error ?? walletQuery.error} /> : null}
      {!key ? <EmptyState title='Sub-key not found' description='The requested key is unavailable for the selected project.' /> : null}
      {key ? (
        <div className='space-y-6'>
          <PageHeader
            title={key.name}
            description={`${key.productName} for ${key.projectName}. Last used ${formatDateTime(key.lastUsedAt)}.`}
            actions={
              <>
                <Button asChild><a href='/project/relay-subkeys/verify'>Verify setup</a></Button>
                <Button variant='outline' asChild><a href='/project/relay-subkeys/get-started'>Open guide</a></Button>
              </>
            }
          />
          <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
            <MetricCard label='Key status' value={key.status} hint='Persisted state controls whether runtime accepts requests.' />
            <MetricCard label='Balance' value={wallet ? formatCurrency(wallet.availableAmount, wallet.currency) : '-'} hint='Low balance can block otherwise valid setup.' />
            <MetricCard label='Today requests' value={formatNumber(key.usage.todayRequests)} hint={`${formatNumber(key.usage.todayTokens)} tokens today`} />
            <MetricCard label='Monthly cost' value={formatCurrency(key.usage.monthlyCost)} hint={`Limit ${formatCurrency(key.limits.monthlyCostLimit)}`} />
          </div>
          <Tabs defaultValue='credential'>
            <TabsList>
              <TabsTrigger value='credential'>Credential</TabsTrigger>
              <TabsTrigger value='limits'>Limits</TabsTrigger>
              <TabsTrigger value='requests'>Requests</TabsTrigger>
            </TabsList>
            <TabsContent value='credential' className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
              <Card>
                <CardHeader>
                  <CardTitle>Safe credential values</CardTitle>
                  <CardDescription>Use this information to configure SDKs. Historical plaintext secrets are never shown.</CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div className='rounded-lg border p-4 font-mono text-sm'>{key.maskedKey}</div>
                  <div className='rounded-lg border p-4 font-mono text-sm'>{key.baseUrl}</div>
                  <div className='flex flex-wrap gap-2'>
                    <KeyStatusBadge status={key.status} />
                    <DerivedStateBadges states={key.derivedStates} />
                  </div>
                  {key.usage.recentFailure ? <Alert><AlertTitle>Recent failure</AlertTitle><AlertDescription>{key.usage.recentFailure}</AlertDescription></Alert> : null}
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>Integration checklist</CardTitle>
                  <CardDescription>Confirm these values before running a verification request.</CardDescription>
                </CardHeader>
                <CardContent className='space-y-2 text-sm'>
                  <div>Authorization: Bearer relay sub-key</div>
                  <div>Base URL: {key.baseUrl}</div>
                  <div>Product: {key.productName}</div>
                  <div>Expiry: {formatDateTime(key.expiresAt)}</div>
                </CardContent>
              </Card>
            </TabsContent>
            <TabsContent value='limits'>
              <div className='grid gap-4 md:grid-cols-4'>
                <MetricCard label='Daily requests' value={formatNumber(key.limits.dailyRequestLimit)} hint='Request cap before quota errors.' />
                <MetricCard label='Daily tokens' value={formatNumber(key.limits.dailyTokenLimit)} hint='Token cap before quota errors.' />
                <MetricCard label='Monthly cost' value={formatCurrency(key.limits.monthlyCostLimit)} hint='Cost cap for the key wallet.' />
                <MetricCard label='Concurrency' value={formatNumber(key.limits.concurrencyLimit)} hint='Maximum in-flight relay calls.' />
              </div>
            </TabsContent>
            <TabsContent value='requests'>
              <RequestTraceTable requests={requests} />
            </TabsContent>
          </Tabs>
        </div>
      ) : null}
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysUsagePage() {
  const usageQuery = useProjectRelayUsageQuery();

  return (
    <ProjectShell title='Usage and Billing' description='Review project-scoped balance, ledger, and request usage without operator-only adjustment controls.'>
      {usageQuery.isLoading ? <LoadingCards /> : null}
      {usageQuery.error ? <ErrorState error={usageQuery.error} /> : null}
      {usageQuery.data ? (
        <div className='space-y-6'>
          <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
            <MetricCard label='Wallets' value={formatNumber(usageQuery.data.wallets.length)} hint='Visible project relay wallets.' />
            <MetricCard label='Available balance' value={formatCurrency(usageQuery.data.wallets.reduce((sum, wallet) => sum + wallet.availableAmount, 0))} hint='Aggregate available balance.' />
            <MetricCard label='Requests' value={formatNumber(usageQuery.data.usage.reduce((sum, day) => sum + day.requests, 0))} hint='Requests in the visible usage window.' />
            <MetricCard label='Cost' value={formatCurrency(usageQuery.data.usage.reduce((sum, day) => sum + day.totalCost, 0))} hint='Usage settlement cost in the visible window.' />
          </div>
          <TableFrame title='Daily usage' description='Project-scoped request, token, and cost summary.'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead className='text-right'>Requests</TableHead>
                  <TableHead className='text-right'>Tokens</TableHead>
                  <TableHead className='text-right'>Cost</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usageQuery.data.usage.map((day) => (
                  <TableRow key={`${day.relayKeyId}-${day.statDate}`}>
                    <TableCell>{day.statDate}</TableCell>
                    <TableCell>{day.relayKeyId}</TableCell>
                    <TableCell className='text-right'>{formatNumber(day.requests)}</TableCell>
                    <TableCell className='text-right'>{formatNumber(day.promptTokens + day.completionTokens)}</TableCell>
                    <TableCell className='text-right'>{formatCurrency(day.totalCost)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableFrame>
          <TableFrame title='Wallet ledger' description='Recharge, usage charge, refund, and adjustment records are read-only here.'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Time</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Note</TableHead>
                  <TableHead>Reference</TableHead>
                  <TableHead className='text-right'>Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usageQuery.data.ledgerEntries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{formatDateTime(entry.createdAt)}</TableCell>
                    <TableCell><StatusBadge variant={ledgerTypeVariant(entry.type)}>{entry.type}</StatusBadge></TableCell>
                    <TableCell className='whitespace-normal'>{entry.note}</TableCell>
                    <TableCell>{entry.referenceId ?? '-'}</TableCell>
                    <TableCell className='text-right'>{formatCurrency(entry.amount, entry.currency)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableFrame>
          <RequestTraceTable requests={usageQuery.data.recentRequests} />
        </div>
      ) : null}
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysGetStartedPage() {
  const keysQuery = useProjectRelayKeysQuery();
  const primaryKey = keysQuery.data?.[0];
  const baseUrl = primaryKey?.baseUrl ?? 'https://axonhub.example.com/v1';
  const snippet = `import OpenAI from 'openai';\n\nconst client = new OpenAI({\n  apiKey: process.env.AXONHUB_RELAY_KEY,\n  baseURL: '${baseUrl}',\n});\n\nconst response = await client.chat.completions.create({\n  model: '${primaryKey?.productName.includes('Claude') ? 'claude-3-5-haiku-latest' : 'gpt-4.1-mini'}',\n  messages: [{ role: 'user', content: 'Hello from AxonHub Relay' }],\n});`;

  return (
    <ProjectShell title='Get Started' description='Configure existing compatible SDKs to use AxonHub relay sub-keys instead of upstream provider keys.'>
      <div className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
        <Card>
          <CardHeader>
            <CardTitle>Quickstart</CardTitle>
            <CardDescription>Use the AxonHub base URL and the one-time relay sub-key value provided at issuance.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <Textarea value={snippet} readOnly className='min-h-72 font-mono text-sm' />
            <Alert>
              <AlertTitle>Credential ownership</AlertTitle>
              <AlertDescription>Relay sub-keys are AxonHub-issued credentials. They should never be confused with raw upstream provider secrets.</AlertDescription>
            </Alert>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Checklist</CardTitle>
            <CardDescription>Most setup failures come from the wrong base URL, expired key, depleted balance, or unsupported model.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3 text-sm'>
            <div className='rounded-lg bg-muted/40 p-3'>1. Select an active relay key from the project keys page.</div>
            <div className='rounded-lg bg-muted/40 p-3'>2. Set SDK base URL to {baseUrl}.</div>
            <div className='rounded-lg bg-muted/40 p-3'>3. Pick a model supported by the bound product.</div>
            <div className='rounded-lg bg-muted/40 p-3'>4. Run the verify page and review request trace feedback.</div>
          </CardContent>
        </Card>
      </div>
    </ProjectShell>
  );
}

export function ProjectRelaySubkeysVerifyPage() {
  const keysQuery = useProjectRelayKeysQuery();
  const [selectedKeyId, setSelectedKeyId] = useState('');
  const [model, setModel] = useState('gpt-4.1-mini');
  const selectedKey = keysQuery.data?.find((key) => key.id === selectedKeyId) ?? keysQuery.data?.[0];
  const balanceRisk = selectedKey?.derivedStates.includes('low_balance') ?? false;
  const quotaRisk = selectedKey?.derivedStates.includes('quota_reached') ?? false;
  const poolRisk = selectedKey?.derivedStates.includes('upstream_pool_degraded') ?? false;
  const ready = Boolean(selectedKey && selectedKey.status === 'active' && !balanceRisk && !quotaRisk);

  return (
    <ProjectShell title='Verify Integration' description='Run a dry readiness review before sending traffic through a relay sub-key.'>
      {keysQuery.isLoading ? <LoadingCards /> : null}
      {keysQuery.error ? <ErrorState error={keysQuery.error} /> : null}
      {keysQuery.data ? (
        <div className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
          <Card>
            <CardHeader>
              <CardTitle>Verification inputs</CardTitle>
              <CardDescription>The MVP performs a frontend readiness check and does not issue network traffic.</CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <Select value={selectedKey?.id ?? selectedKeyId} onValueChange={setSelectedKeyId}>
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder='Select key' />
                </SelectTrigger>
                <SelectContent>
                  {keysQuery.data.map((key) => (
                    <SelectItem key={key.id} value={key.id}>
                      {key.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input value={model} onChange={(event) => setModel(event.target.value)} placeholder='Model to test' />
              <div className='rounded-lg border p-4 font-mono text-sm'>{selectedKey?.baseUrl ?? 'Select a key to see the relay base URL'}</div>
              <Button disabled={!selectedKey}>Run frontend readiness check</Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Readiness result</CardTitle>
              <CardDescription>These checks mirror the integration verify flow without touching backend runtime endpoints.</CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              {!selectedKey ? <EmptyState title='No key selected' description='Select an issued key to evaluate setup readiness.' /> : null}
              {selectedKey ? (
                <>
                  <Alert variant={ready ? 'default' : 'destructive'}>
                    <AlertTitle>{ready ? 'Ready for a live verification call' : 'Resolve readiness issues first'}</AlertTitle>
                    <AlertDescription>
                      {ready
                        ? `Key ${selectedKey.name} is active for ${model}. Use request traces after your first live call.`
                        : 'The selected key has status, balance, quota, or shared-pool signals that may block runtime traffic.'}
                    </AlertDescription>
                  </Alert>
                  <div className='grid gap-2 text-sm'>
                    <div className='flex items-center justify-between rounded-lg bg-muted/40 p-3'><span>Status</span><KeyStatusBadge status={selectedKey.status} /></div>
                    <div className='flex items-center justify-between rounded-lg bg-muted/40 p-3'><span>Balance risk</span><Badge variant={balanceRisk ? 'destructive' : 'default'}>{balanceRisk ? 'risk' : 'ok'}</Badge></div>
                    <div className='flex items-center justify-between rounded-lg bg-muted/40 p-3'><span>Quota risk</span><Badge variant={quotaRisk ? 'destructive' : 'default'}>{quotaRisk ? 'risk' : 'ok'}</Badge></div>
                    <div className='flex items-center justify-between rounded-lg bg-muted/40 p-3'><span>Pool health</span><Badge variant={poolRisk ? 'secondary' : 'default'}>{poolRisk ? 'degraded' : 'ok'}</Badge></div>
                  </div>
                </>
              ) : null}
            </CardContent>
          </Card>
        </div>
      ) : null}
    </ProjectShell>
  );
}
