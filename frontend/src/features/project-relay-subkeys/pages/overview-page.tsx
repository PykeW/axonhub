import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysOverviewPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Project Relay Subkeys'
      description='Skeleton landing page for buyer-facing relay products, sub-key access, and onboarding flows.'
      actions={[
        { label: 'Browse Products' },
        { label: 'Open Usage View', variant: 'secondary' },
      ]}
      summary={[
        { label: 'Visible products', value: '--', hint: 'Product availability will come from the project-scoped relay catalog.' },
        { label: 'Issued sub-keys', value: '--', hint: 'Assigned keys will surface once relay issuance APIs are connected.' },
        { label: 'Balance snapshot', value: '--', hint: 'Usage and wallet data stay read-only in this project-facing skeleton.' },
        { label: 'Verification state', value: 'Pending', hint: 'Live verification remains disabled until backend endpoints are ready.' },
      ]}
      sections={[
        {
          title: 'MVP entry points',
          description: 'This overview keeps the project-facing routes discoverable without adding business logic.',
          bullets: [
            'Products route will explain which shared-capacity bundles the current project can use.',
            'Keys routes will focus on masked credentials, status badges, and recent request hints.',
            'Usage and verify routes reserve space for consumption visibility and integration checks.',
          ],
          badge: 'Thin route target',
        },
        {
          title: 'What is intentionally missing',
          description: 'The skeleton avoids speculative data handling and purchase workflows.',
          bullets: [
            'No API requests, forms, or mutations are wired in this pass.',
            'No fake balances, models, or request histories are generated.',
            'No operator-only controls are exposed in the project subtree.',
          ],
          footer: 'Follow-up work can replace these placeholders with loaders and query-backed components.',
        },
      ]}
      asideTitle='Implementation notes'
      asideDescription='This page is meant to stay stable while backend contracts are finalized.'
      asideBullets={[
        'Project routes live only under the authenticated project subtree.',
        'The route file stays thin and delegates all page structure to feature code.',
        'No route-permission wiring is added because the current skeleton does not require it.',
      ]}
    />
  );
}
