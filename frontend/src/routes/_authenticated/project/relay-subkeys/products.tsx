import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysProductsPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysProducts() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysProductsPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/products')({
  component: ProtectedProjectRelaySubkeysProducts,
});
