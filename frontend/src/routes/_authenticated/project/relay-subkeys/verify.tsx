import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysVerifyPage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysVerify() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysVerifyPage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/verify')({
  component: ProtectedProjectRelaySubkeysVerify,
});
