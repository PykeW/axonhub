import { createFileRoute } from '@tanstack/react-router';
import { RelayRequestListPage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/requests/')({
  component: RelayRequestListPage,
});
