import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysOverviewPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysOverview() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysOverviewPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/')({
  component: ProtectedProjectRelaySubkeysOverview,
});
