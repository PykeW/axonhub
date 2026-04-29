import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysKeyDetailPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysKeyDetail() {
  const { keyId } = Route.useParams();

  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys', 'read_requests']}>
        <ProjectRelaySubkeysKeyDetailPage keyId={keyId} />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/keys/$keyId')({
  component: ProtectedProjectRelaySubkeysKeyDetail,
});
