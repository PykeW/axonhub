import { createFileRoute } from '@tanstack/react-router';
import { ProjectGuard } from '@/components/project-guard';
import { ProjectRelaySubkeysUsagePage } from '@/features/project-relay-subkeys';

function ProtectedProjectRelaySubkeysUsage() {
  return (
    <ProjectGuard>
      <ProjectRelaySubkeysUsagePage />
    </ProjectGuard>
  );
}

export const Route = createFileRoute('/_authenticated/project/relay-subkeys/usage')({
  component: ProtectedProjectRelaySubkeysUsage,
});
