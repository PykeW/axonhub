import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { RouteGuard } from '@/components/route-guard';
import { ProjectRelaySubkeysProductsPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysProducts() {
  return (
    <ProjectGuard>
      <RouteGuard requiredScopes={['read_api_keys', 'read_requests']}>
        <ProjectRelaySubkeysProductsPage />
      </RouteGuard>
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/products')({
  component: ProtectedProjectRelaySubkeysProducts,
});
