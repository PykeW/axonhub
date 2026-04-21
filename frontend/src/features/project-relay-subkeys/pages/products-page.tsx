import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysProductsPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Relay Products'
      description='Placeholder catalog view for shared-capacity products available to the current project.'
      actions={[
        { label: 'Review Models' },
        { label: 'Contact Operator', variant: 'outline' },
      ]}
      summary={[
        { label: 'Catalog status', value: 'Placeholder', hint: 'Product rows will render once project-visible relay products are queryable.' },
        { label: 'Model coverage', value: '--', hint: 'Allowed model lists will be surfaced from product metadata.' },
        { label: 'Billing mode', value: '--', hint: 'Prepaid and shared-pool rules stay informational on the project side.' },
        { label: 'Activation gate', value: 'Operator managed', hint: 'Project users can inspect product scope but cannot activate products.' },
      ]}
      sections={[
        {
          title: 'Catalog cards planned for MVP',
          description: 'Each product card will emphasize compatibility and guardrails before any purchase flow exists.',
          bullets: [
            'Product name, provider family, and supported model ranges.',
            'Shared-capacity notes such as timeout defaults and usage constraints.',
            'Links into the key list so a project admin can verify which issued sub-keys map to each product.',
          ],
          badge: 'Project read-only',
        },
        {
          title: 'Operator handoff signals',
          description: 'The UI reserves room for explaining when a project needs help from the platform operator.',
          bullets: [
            'Missing product access should point users back to an operator-issued entitlement flow.',
            'Pool degradation or maintenance windows should be explained here without exposing operator controls.',
            'Future loaders can attach product health or availability messages without changing the route shape.',
          ],
        },
      ]}
      asideTitle='Scope guard'
      asideDescription='This page intentionally avoids shopping-cart or payment behavior.'
      asideBullets={[
        'No self-service checkout is included in the MVP skeleton.',
        'No editable product settings appear in the project-facing experience.',
        'Cards and badges are placeholders only until real project catalog data lands.',
      ]}
    />
  );
}
