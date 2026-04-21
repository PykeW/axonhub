import { createFileRoute } from '@tanstack/react-router';
import { RelayKeyCreatePage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/create')({
  component: RelayKeyCreatePage,
});
