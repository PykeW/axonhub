import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysKeyDetailPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Sub-Key Detail'
      description='Placeholder detail page for masked credentials, status explanation, and recent relay request visibility.'
      actions={[
        { label: 'Copy API Key' },
        { label: 'Copy Base URL', variant: 'secondary' },
        { label: 'Verify Integration', variant: 'outline' },
      ]}
      summary={[
        { label: 'Key status', value: 'Unknown', hint: 'Persisted state and derived badges will render here once loaders are connected.' },
        { label: 'Bound product', value: '--', hint: 'The associated relay product will explain supported models and limits.' },
        { label: 'Expiry', value: '--', hint: 'Expiry and low-balance warnings should remain visible even without operator actions.' },
        { label: 'Recent failures', value: '--', hint: 'Project users will be able to inspect recent request outcomes without opening operator tooling.' },
      ]}
      sections={[
        {
          title: 'Credential block',
          description: 'The detail page will eventually present onboarding values in a copy-first layout.',
          bullets: [
            'Masked API key and project-safe copy affordances.',
            'AxonHub relay base URL with compatibility notes for supported SDKs.',
            'Supported model hints and shared-capacity caveats taken from the bound product.',
          ],
          badge: 'Read-only MVP',
        },
        {
          title: 'Health and troubleshooting',
          description: 'The skeleton reserves space for explainable request outcomes and key status guidance.',
          bullets: [
            'Status badge area for active, suspended, exhausted, expired, and degraded pool hints.',
            'Recent request snippets that can explain whether failures come from balance, quota, or upstream pool issues.',
            'Usage and verification links so the detail page remains the hub for project onboarding.',
          ],
          footer: 'Operator-only actions such as recharge, suspend, or archive intentionally remain absent.',
        },
      ]}
      asideTitle='Integration reminder'
      asideDescription='This view should teach teams how to use an AxonHub downstream credential safely.'
      asideBullets={[
        'Treat the sub-key as an AxonHub-issued credential, not a raw provider key.',
        'Future copy actions can become active without changing the overall layout.',
        'Project-facing troubleshooting must stay explanatory instead of operational.',
      ]}
    />
  );
}
