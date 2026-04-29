import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysUsagePage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysUsage() {
  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys', 'read_requests']}>
        <ProjectRelaySubkeysUsagePage />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/usage')({
  component: ProtectedProjectRelaySubkeysUsage,
});
