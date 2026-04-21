import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysUsagePage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Usage and Billing'
      description='Placeholder reporting page for balance snapshots, usage trends, and recent relay charge activity.'
      actions={[
        { label: 'Refresh Snapshot' },
        { label: 'Open Key Detail', variant: 'secondary' },
      ]}
      summary={[
        { label: 'Available balance', value: '--', hint: 'Wallet data stays informational for project users in the MVP scope.' },
        { label: 'Daily usage', value: '--', hint: 'Aggregate request counts and token metrics can populate this view later.' },
        { label: 'Monthly cost', value: '--', hint: 'Billing rollups should remain project-scoped and read-only.' },
        { label: 'Ledger entries', value: '--', hint: 'Recent recharge or charge events can appear here when backend support is ready.' },
      ]}
      sections={[
        {
          title: 'Reporting placeholders',
          description: 'The layout reserves separate panels for snapshot metrics and recent charge evidence.',
          bullets: [
            'Trend cards for requests, tokens, and spend over a selected period.',
            'Ledger timeline for recharge, usage settlement, and manual adjustments that affect the project balance.',
            'Cross-links back to key details when users need per-key context.',
          ],
          badge: 'Analytics shell',
        },
        {
          title: 'MVP boundaries',
          description: 'This page avoids export and reconciliation workflows while still protecting the route shape.',
          bullets: [
            'No CSV export, invoice workflows, or payment actions are introduced.',
            'No editable billing preferences are shown in the project-facing area.',
            'Placeholders can be replaced by charts or tables later without touching route wiring.',
          ],
        },
      ]}
      asideTitle='Why this page exists'
      asideDescription='Project admins need a single place to understand shared-capacity cost and balance behavior.'
      asideBullets={[
        'Usage visibility reduces confusion when keys enter low-balance or quota-reached states.',
        'The project subtree can expose settlement facts without exposing operator ledger controls.',
        'This skeleton keeps the reporting route ready for future query integration.',
      ]}
    />
  );
}
