import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysKeyDetailPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysKeyDetail() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysKeyDetailPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/keys/$keyId')({
  component: ProtectedProjectRelaySubkeysKeyDetail,
});
