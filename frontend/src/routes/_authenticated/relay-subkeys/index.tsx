import { createFileRoute } from '@tanstack/react-router';
import { RelaySubkeysOverviewPage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/')({
  component: RelaySubkeysOverviewPage,
});
