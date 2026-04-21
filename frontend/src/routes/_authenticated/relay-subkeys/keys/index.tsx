import { createFileRoute } from '@tanstack/react-router';
import { RelayKeyListPage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/')({
  component: RelayKeyListPage,
});
