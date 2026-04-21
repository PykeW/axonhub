import { createFileRoute } from '@tanstack/react-router';
import { RelayProductCreatePage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/create')({
  component: RelayProductCreatePage,
});
