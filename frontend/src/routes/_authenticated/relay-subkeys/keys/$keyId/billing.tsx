import { createFileRoute } from '@tanstack/react-router';
import { RelayKeyBillingPage } from '@/features/relay-subkeys';

function RelayKeyBillingRoute() {
  const { keyId } = Route.useParams();

  return <RelayKeyBillingPage keyId={keyId} />;
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/$keyId/billing')({
  component: RelayKeyBillingRoute,
});
