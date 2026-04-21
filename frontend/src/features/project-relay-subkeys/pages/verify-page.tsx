import { ProjectRelaySubkeysPageShell } from '../components/page-shell';

export function ProjectRelaySubkeysVerifyPage() {
  return (
    <ProjectRelaySubkeysPageShell
      title='Verify Integration'
      description='Placeholder verification page for confirming relay endpoint setup and interpreting project-side readiness states.'
      actions={[
        { label: 'Run Verification' },
        { label: 'Back to Get Started', variant: 'secondary' },
      ]}
      summary={[
        { label: 'Endpoint check', value: 'Not started', hint: 'A focused verification widget can be introduced once project-safe test APIs exist.' },
        { label: 'Credential check', value: 'Not started', hint: 'Sub-key validation remains intentionally inactive in the compile-safe skeleton.' },
        { label: 'Model check', value: 'Not started', hint: 'Supported-model confirmation can be derived from product metadata later.' },
        { label: 'Request trace', value: 'Unavailable', hint: 'Real-time verification should link to project request history after APIs are available.' },
      ]}
      sections={[
        {
          title: 'Planned verification flow',
          description: 'This route reserves space for a narrow project-safe verification experience.',
          bullets: [
            'Step-by-step endpoint, credential, and model checks.',
            'Result messages that explain whether failures come from setup, quota, or upstream shared-pool conditions.',
            'Links back to onboarding and key detail pages for next actions.',
          ],
          badge: 'Future interactive',
        },
        {
          title: 'Current skeleton behavior',
          description: 'The page compiles without inventing network behavior or fake success signals.',
          bullets: [
            'Buttons remain disabled and do not trigger any API traffic.',
            'Messaging stays descriptive so QA and future implementers know the intended flow.',
            'The route can later host a small verification form without disrupting project navigation.',
          ],
        },
      ]}
      asideTitle='Guardrail'
      asideDescription='Verification should remain project-safe and avoid leaking operator diagnostics.'
      asideBullets={[
        'Keep the experience focused on setup confirmation, not shared-pool operations.',
        'Use explainable states rather than silent failures when real APIs are added.',
        'Leave operator-side request debugging to the separate operator ownership area.',
      ]}
    />
  );
}
