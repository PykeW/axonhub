import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysKeysPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysKeys() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysKeysPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/keys/')({
  component: ProtectedProjectRelaySubkeysKeys,
});
