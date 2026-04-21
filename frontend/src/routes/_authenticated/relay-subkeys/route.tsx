import { createFileRoute } from '@tanstack/react-router';
import { RelaySubkeysLayout } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys')({
  component: RelaySubkeysLayout,
});
