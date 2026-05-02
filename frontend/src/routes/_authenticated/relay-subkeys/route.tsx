import { createFileRoute } from '@tanstack/react-router';
import { RelaySubkeysLayout } from '@/features/relay-subkeys';

// Legacy operator-managed Relay/Sub-Key console kept for historical compatibility.
// New Share/Use UX should live under /share and /use until this route tree can be removed.
export const Route = createFileRoute('/_authenticated/relay-subkeys')({
  component: RelaySubkeysLayout,
});
