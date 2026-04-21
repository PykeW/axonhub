import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysGetStartedPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Get Started'
      description='Placeholder onboarding page for using AxonHub relay sub-keys with existing OpenAI-compatible client flows.'
      actions={[
        { label: 'Copy Quickstart Snippet' },
        { label: 'Open Verify Page', variant: 'secondary' },
      ]}
      summary={[
        { label: 'Setup mode', value: 'SDK compatible', hint: 'This route will host copy-ready examples for existing client integrations.' },
        { label: 'Credential type', value: 'AxonHub sub-key', hint: 'The messaging should clearly distinguish relay keys from upstream provider secrets.' },
        { label: 'Base URL', value: '--', hint: 'Project-scoped relay base URLs will land here once environment data is available.' },
        { label: 'Error guide', value: 'Planned', hint: 'Common onboarding failures can be explained without embedding real API calls today.' },
      ]}
      sections={[
        {
          title: 'Quickstart outline',
          description: 'The final page can map directly to the documentation-described integration journey.',
          bullets: [
            'SDK snippets for OpenAI-compatible clients using AxonHub base URLs.',
            'Model selection notes driven by the bound relay product configuration.',
            'Clear messaging about one-time credential copy expectations and downstream relay ownership.',
          ],
          badge: 'Docs-backed',
        },
        {
          title: 'Failure guidance placeholders',
          description: 'Explainable onboarding help is more important than interactivity in this initial shell.',
          bullets: [
            'What to check when authentication fails or the wrong base URL is used.',
            'How low balance, expiry, or exhausted quota may appear to project users.',
            'When to move to the verify page versus when to contact an operator.',
          ],
        },
      ]}
      asideTitle='Copy strategy'
      asideDescription='The eventual page can swap placeholders for code blocks and callouts without changing the shell.'
      asideBullets={[
        'Keep language focused on compatibility, not provider-native setup.',
        'Avoid hardcoded example secrets or fake request payloads in the skeleton.',
        'The verify route should complement this page instead of duplicating every instruction.',
      ]}
    />
  );
}
