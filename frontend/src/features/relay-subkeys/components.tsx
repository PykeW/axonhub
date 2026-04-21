import { Outlet } from '@tanstack/react-router';
import { Header } from '@/components/layout/header';
import { Main } from '@/components/layout/main';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

type BadgeTone = 'default' | 'secondary' | 'outline';

interface SummaryItem {
  title: string;
  value: string;
  note: string;
}

interface PlaceholderSectionProps {
  title: string;
  description: string;
  bullets: string[];
  actions?: string[];
  badges?: Array<{ label: string; tone?: BadgeTone }>;
}

interface DetailPageProps {
  productId?: string;
  keyId?: string;
}

const operatorAreas = [
  'Products',
  'Product creation',
  'Sub-key inventory',
  'Sub-key issuance',
  'Request troubleshooting',
  'Channel pool health',
];

const overviewItems: SummaryItem[] = [
  {
    title: 'Products',
    value: 'List + create + detail',
    note: 'Scaffolds the operator flow for shared-capacity products and channel pool setup.',
  },
  {
    title: 'Keys',
    value: 'List + create + detail',
    note: 'Keeps issuance, lifecycle controls, and one-time credential guidance in view.',
  },
  {
    title: 'Troubleshooting',
    value: 'Requests + pool health',
    note: 'Separates key-level diagnostics from product-level upstream capacity checks.',
  },
];

function SummaryGrid({ items }: { items: SummaryItem[] }) {
  return (
    <div className='grid gap-4 md:grid-cols-3'>
      {items.map((item) => (
        <Card key={item.title}>
          <CardHeader className='gap-1'>
            <CardDescription>{item.title}</CardDescription>
            <CardTitle className='text-base'>{item.value}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className='text-sm text-muted-foreground'>{item.note}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function PlaceholderSection({ title, description, bullets, actions, badges }: PlaceholderSectionProps) {
  return (
    <Card>
      <CardHeader className='gap-3'>
        <div className='flex flex-wrap items-center gap-2'>
          <CardTitle className='text-base'>{title}</CardTitle>
          {badges?.map((badge) => (
            <Badge key={badge.label} variant={badge.tone ?? 'secondary'}>
              {badge.label}
            </Badge>
          ))}
        </div>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <ul className='list-disc space-y-2 pl-5 text-sm text-muted-foreground'>
          {bullets.map((bullet) => (
            <li key={bullet}>{bullet}</li>
          ))}
        </ul>
        {actions && actions.length > 0 ? (
          <div className='flex flex-wrap gap-2'>
            {actions.map((action) => (
              <Button key={action} type='button' variant='outline' size='sm'>
                {action}
              </Button>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function PageIntro({ title, description, badge }: { title: string; description: string; badge?: string }) {
  return (
    <div className='flex flex-wrap items-start justify-between gap-3'>
      <div className='space-y-1'>
        <h3 className='text-lg font-semibold tracking-tight'>{title}</h3>
        <p className='text-sm text-muted-foreground'>{description}</p>
      </div>
      {badge ? <Badge variant='outline'>{badge}</Badge> : null}
    </div>
  );
}

export function RelaySubkeysLayout() {
  return (
    <>
      <Header fixed>
        <div className='flex flex-1 items-center justify-between gap-4'>
          <div>
            <h2 className='text-xl font-bold tracking-tight'>Relay Subkeys</h2>
            <p className='text-sm text-muted-foreground'>
              Operator-facing skeleton routes for shared-capacity products, key issuance, and troubleshooting.
            </p>
          </div>
          <Badge variant='secondary'>MVP skeleton</Badge>
        </div>
      </Header>

      <Main fixed className='space-y-6 overflow-y-auto'>
        <Card>
          <CardHeader>
            <CardTitle>Operator scope</CardTitle>
            <CardDescription>
              This route group intentionally stays isolated from project-facing relay-subkey pages while the operator MVP is scaffolded.
            </CardDescription>
          </CardHeader>
          <CardContent className='flex flex-wrap gap-2'>
            {operatorAreas.map((area) => (
              <Badge key={area} variant='outline'>
                {area}
              </Badge>
            ))}
          </CardContent>
        </Card>

        <Outlet />
      </Main>
    </>
  );
}

export function RelaySubkeysOverviewPage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Relay-subkey operations overview'
        description='Landing page for the operator workspace. The cards below mark the primary flows that still need API and interaction wiring.'
      />
      <SummaryGrid items={overviewItems} />
      <PlaceholderSection
        title='MVP coverage in this skeleton'
        description='The scaffold mirrors the page flow documentation without inventing backend behavior.'
        bullets={[
          'Products cover creation, detail, and channel-pool planning surfaces.',
          'Keys cover issuance, lifecycle controls, masked credential messaging, and recovery paths.',
          'Troubleshooting covers request traces and upstream pool health as separate operator views.',
        ]}
        actions={['Review products', 'Review keys', 'Open troubleshooting views']}
        badges={[{ label: 'No API wiring' }, { label: 'Compile-oriented', tone: 'outline' }]}
      />
    </div>
  );
}

export function RelayProductListPage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Shared-capacity products'
        description='Placeholder inventory for sellable relay products and their current operational posture.'
      />
      <SummaryGrid
        items={[
          {
            title: 'Draft products',
            value: 'Placeholder count',
            note: 'Later wiring can split draft, active, and paused products without changing the route shape.',
          },
          {
            title: 'Channel coverage',
            value: 'Pending data',
            note: 'Healthy versus degraded upstream pools will be surfaced once product-channel queries exist.',
          },
          {
            title: 'Follow-up',
            value: 'Create + detail',
            note: 'Operators will branch into create and detail routes from this surface.',
          },
        ]}
      />
      <PlaceholderSection
        title='List expectations'
        description='The final list should help operators identify which products are ready to issue keys against.'
        bullets={[
          'Show product code, provider type, allowed models, and activation state.',
          'Highlight pool health so an empty or degraded upstream set is obvious before issuing keys.',
          'Keep transitions to create and detail pages visible without embedding edit logic here yet.',
        ]}
        actions={['Create product', 'Inspect product detail']}
        badges={[{ label: 'Operator page' }]}
      />
    </div>
  );
}

export function RelayProductCreatePage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Create relay product'
        description='Form skeleton for defining product metadata before channel-pool binding is added.'
        badge='Draft flow'
      />
      <PlaceholderSection
        title='Intended first-step inputs'
        description='The MVP doc calls for operators to create the product record first and bind channels afterward.'
        bullets={[
          'Product code and display name.',
          'Provider type, allowed model scope, and default timeout.',
          'Billing mode and any operator-facing notes needed before activation.',
        ]}
        actions={['Save draft', 'Cancel']}
        badges={[{ label: 'No persistence yet' }, { label: 'Metadata first', tone: 'outline' }]}
      />
      <PlaceholderSection
        title='Next step after save'
        description='Channel-pool configuration should remain a follow-up step instead of being hard-coded into creation.'
        bullets={[
          'Land on product detail after successful creation.',
          'Guide the operator toward channel selection, priority, and fallback settings.',
          'Prevent activation until the bound pool is healthy enough for traffic.',
        ]}
      />
    </div>
  );
}

export function RelayProductDetailPage({ productId }: DetailPageProps) {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Product detail and channel pool'
        description='Route placeholder for product metadata, bound channels, and model-scope review.'
        badge={productId ? `Product ${productId}` : 'Product detail'}
      />
      <PlaceholderSection
        title='Overview tab placeholder'
        description='This area will summarize the product posture before operators make pool changes.'
        bullets={[
          'Product code, provider type, activation state, and model allowances.',
          'Default timeout and billing mode for downstream keys issued from this product.',
          'Counts for total bound channels and currently healthy candidates.',
        ]}
        actions={['Edit metadata', 'Change status']}
        badges={[{ label: 'Detail route' }]}
      />
      <PlaceholderSection
        title='Channel-pool placeholder'
        description='The MVP specifically calls out binding channels after product creation.'
        bullets={[
          'List each bound channel with priority, weight, fallback behavior, and model filters.',
          'Expose degradation reasons so operators can distinguish pool issues from product misconfiguration.',
          'Leave room for assigned-key visibility without coupling the skeleton to backend responses yet.',
        ]}
        actions={['Bind channel', 'Reorder priority', 'Restrict models']}
        badges={[{ label: 'Pool config', tone: 'outline' }]}
      />
    </div>
  );
}

export function RelayKeyListPage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Relay sub-key inventory'
        description='Operator view for issued downstream keys, their lifecycle, and lightweight balance signals.'
      />
      <SummaryGrid
        items={[
          {
            title: 'Status focus',
            value: 'Active / suspended / exhausted',
            note: 'Persistent key states can later be combined with runtime badges like expiry and low balance.',
          },
          {
            title: 'Owner lookup',
            value: 'Project-centric',
            note: 'Operators should be able to search by project and product once data is connected.',
          },
          {
            title: 'Primary actions',
            value: 'Issue or inspect',
            note: 'Creation and detail routes stay separate to keep this listing page thin.',
          },
        ]}
      />
      <PlaceholderSection
        title='List expectations'
        description='This page should help support and operators spot customer-impacting issues quickly.'
        bullets={[
          'Surface masked key, owning project, bound product, and latest lifecycle status.',
          'Reserve space for low-balance, expired, and upstream-pool-degraded badges.',
          'Keep the page focused on triage and navigation instead of embedding lifecycle mutations here.',
        ]}
        actions={['Create sub-key', 'Open key detail']}
        badges={[{ label: 'Support-friendly' }]}
      />
    </div>
  );
}

export function RelayKeyCreatePage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Issue relay sub-key'
        description='Placeholder issuance flow for binding a project to a relay product and defining initial limits.'
        badge='One-time credential flow'
      />
      <PlaceholderSection
        title='Intended create fields'
        description='The final form will collect the minimum information needed for a usable downstream key.'
        bullets={[
          'Target project, selected product, and operator-friendly display name.',
          'Expiry, balance mode, and initial hard-limit settings.',
          'Post-create handoff that reveals the plaintext key exactly once.',
        ]}
        actions={['Issue key', 'Cancel']}
        badges={[{ label: 'No key generation yet' }, { label: 'Plaintext once', tone: 'outline' }]}
      />
    </div>
  );
}

export function RelayKeyDetailPage({ keyId }: DetailPageProps) {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Sub-key detail'
        description='Placeholder detail view for lifecycle controls, balance context, and recent request hints.'
        badge={keyId ? `Key ${keyId}` : 'Key detail'}
      />
      <PlaceholderSection
        title='Overview tab placeholder'
        description='Operators will eventually confirm whether the key is healthy, depleted, expired, or manually paused.'
        bullets={[
          'Masked credential, owning project, bound product, and current persisted status.',
          'Balance snapshot, expiry, last use, and recent failure summary.',
          'Distinct actions for suspend, resume, archive, and post-recharge recovery.',
        ]}
        actions={['Suspend key', 'Resume key', 'Archive key']}
        badges={[{ label: 'Lifecycle controls' }]}
      />
      <PlaceholderSection
        title='Supporting tabs placeholder'
        description='The route is intended to host balance, request, and limit context without inventing tab state yet.'
        bullets={[
          'Balance and ledger details can later be embedded or linked from here.',
          'Recent requests should help operators explain upstream versus balance failures.',
          'Limit settings should remain clearly separated from manual suspension reasons.',
        ]}
        actions={['View ledger', 'Inspect recent requests']}
        badges={[{ label: 'Expandable detail', tone: 'outline' }]}
      />
    </div>
  );
}

export function RelayRequestListPage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Relay request troubleshooting'
        description='Operator trace placeholder for failures by product, key, project, and upstream channel.'
      />
      <PlaceholderSection
        title='Trace expectations'
        description='The final screen should make it easy to explain where a relay request failed in the pipeline.'
        bullets={[
          'Separate auth, key validation, routing, upstream execution, and settlement failure stages.',
          'Show the linked product, key, project, routed channel, and charge outcome.',
          'Leave room for time-range and entity filters without introducing fake datasets now.',
        ]}
        actions={['Filter by product', 'Filter by key', 'Inspect settlement result']}
        badges={[{ label: 'Diagnostics' }, { label: 'No data source yet', tone: 'outline' }]}
      />
    </div>
  );
}

export function RelayChannelPoolHealthPage() {
  return (
    <div className='space-y-6'>
      <PageIntro
        title='Channel pool health'
        description='Operator dashboard placeholder for detecting shared upstream capacity risk by product.'
      />
      <SummaryGrid
        items={[
          {
            title: 'Healthy channels',
            value: 'Pending telemetry',
            note: 'Channel health will later be tied to bound-product views instead of stand-alone channel data only.',
          },
          {
            title: 'Degraded pools',
            value: 'Pending telemetry',
            note: 'Operators need a quick signal when a product has too few viable upstream candidates.',
          },
          {
            title: 'Mitigations',
            value: 'Priority + pause',
            note: 'Common responses include reprioritizing channels, pausing bad ones, or constraining model scope.',
          },
        ]}
      />
      <PlaceholderSection
        title='Dashboard expectations'
        description='This page complements request troubleshooting by focusing on shared-pool risk rather than individual keys.'
        bullets={[
          'Summarize products whose bound channels are unhealthy or exhausted.',
          'Help operators decide whether to pause a product or reweight a healthier channel.',
          'Avoid blaming customer keys when the issue is shared upstream capacity.',
        ]}
        actions={['Review affected products', 'Inspect request traces']}
        badges={[{ label: 'Pool-centric view' }]}
      />
    </div>
  );
}
