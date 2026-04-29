import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysOverviewPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysOverview() {
  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys', 'read_requests']}>
        <ProjectRelaySubkeysOverviewPage />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/')({
  component: ProtectedProjectRelaySubkeysOverview,
});
