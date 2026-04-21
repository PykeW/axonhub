import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysGetStartedPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysGetStarted() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysGetStartedPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/get-started')({
  component: ProtectedProjectRelaySubkeysGetStarted,
});
