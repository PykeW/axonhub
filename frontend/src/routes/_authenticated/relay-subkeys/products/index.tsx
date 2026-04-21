import { createFileRoute } from '@tanstack/react-router';
import { RelayProductListPage } from '@/features/relay-subkeys';

export const Route = createFileRoute('/_authenticated/relay-subkeys/products/')({
  component: RelayProductListPage,
});
