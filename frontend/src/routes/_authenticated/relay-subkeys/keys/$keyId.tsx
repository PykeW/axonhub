import { createFileRoute } from '@tanstack/react-router';
import { RelayKeyDetailPage } from '@/features/relay-subkeys';

function RelayKeyDetailRoute() {
  const { keyId } = Route.useParams();

  return <RelayKeyDetailPage keyId={keyId} />;
}

export const Route = createFileRoute('/_authenticated/relay-subkeys/keys/$keyId')({
  component: RelayKeyDetailRoute,
});
