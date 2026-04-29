import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysVerifyPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysVerify() {
  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys', 'read_requests']}>
        <ProjectRelaySubkeysVerifyPage />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/verify')({
  component: ProtectedProjectRelaySubkeysVerify,
});
