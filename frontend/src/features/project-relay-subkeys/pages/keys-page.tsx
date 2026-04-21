import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysKeysPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Project Sub-Keys'
      description='Placeholder list page for issued relay sub-keys, status badges, and copy-ready onboarding details.'
      actions={[
        { label: 'Copy Base URL' },
        { label: 'View Get Started', variant: 'secondary' },
      ]}
      summary={[
        { label: 'Active keys', value: '--', hint: 'Issued keys will be grouped by project and product when data is wired in.' },
        { label: 'Expiring soon', value: '--', hint: 'Expiry warnings are planned as derived badges on top of persisted key status.' },
        { label: 'Low balance', value: '--', hint: 'Balance warnings will highlight shared-capacity risk before requests fail.' },
        { label: 'Last used', value: '--', hint: 'Recent request activity will surface as read-only metadata in the list.' },
      ]}
      sections={[
        {
          title: 'List layout placeholder',
          description: 'The future list will prioritize safe credential handling over inline actions.',
          bullets: [
            'Masked API key value, bound product, and current key state.',
            'Derived badges for expired, low-balance, or quota-reached conditions.',
            'Links to detail pages that explain onboarding and recent request health.',
          ],
          badge: 'No mutations',
        },
        {
          title: 'Project-facing expectations',
          description: 'The list is intended to remain read-only for MVP so ownership stays clear.',
          bullets: [
            'Projects can inspect issued keys but should not freeze, archive, or recharge them here.',
            'Any rotation or reissuance path can be added later without changing this route path.',
            'Empty states can explain whether keys have not been issued yet or are temporarily unavailable.',
          ],
        },
      ]}
      asideTitle='Credential safety'
      asideDescription='The project list should never reveal operator-only controls or sensitive plaintext secrets.'
      asideBullets={[
        'Plaintext key display is out of scope for this skeleton.',
        'Future detail views can show one-time copy guidance without exposing historical secrets.',
        'List actions stay disabled until exact backend capabilities are available.',
      ]}
    />
  );
}
