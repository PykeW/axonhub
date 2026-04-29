import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysKeysPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysKeys() {
  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys']}>
        <ProjectRelaySubkeysKeysPage />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/keys/')({
  component: ProtectedProjectRelaySubkeysKeys,
});
