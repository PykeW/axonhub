import { Link, createFileRoute, type LinkProps } from '@tanstack/react-router';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

type GuideLink = {
  to: LinkProps['to'];
  title: string;
  description: string;
};

const guideLinks: GuideLink[] = [
  {
    to: '/channels',
    title: 'Open Channels',
    description: 'Upload or manage the provider channels that you own.',
  },
  {
    to: '/models',
    title: 'Review Models',
    description: 'Check model IDs exposed by your channels before sharing them.',
  },
  {
    to: '/project/requests',
    title: 'Watch Requests',
    description: 'Inspect request activity after traffic starts using your channels.',
  },
];

const shareSteps = [
  'Upload or connect the channels you own in Channels.',
  'Keep channels private until they are ready to be reused by others.',
  'Mark channels as shared only when you want them available to the shared pool.',
  'Plan around the MVP refresh quota: each shared channel can refresh about every 5h.',
];

function SharePage() {
  return (
    <div className='mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 md:p-8'>
      <div className='space-y-3'>
        <Badge variant='secondary'>Share MVP</Badge>
        <div className='space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>Share your channels</h1>
          <p className='text-muted-foreground max-w-3xl text-sm md:text-base'>
            Use this guide page to prepare channels you own for private use or for the shared pool. The MVP keeps the real
            channel management in the existing Channels and Models pages.
          </p>
        </div>
      </div>

      <div className='grid gap-4 md:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>Share workflow</CardTitle>
            <CardDescription>Smallest safe path for uploading and managing your own channels.</CardDescription>
          </CardHeader>
          <CardContent>
            <ol className='list-decimal space-y-3 pl-5 text-sm'>
              {shareSteps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Current MVP rules</CardTitle>
            <CardDescription>These labels describe the intended behavior without adding new forms here.</CardDescription>
          </CardHeader>
          <CardContent className='grid gap-3 text-sm'>
            <div className='rounded-lg border p-3'>
              <div className='font-medium'>Private</div>
              <p className='text-muted-foreground'>Only your own routing should use the channel.</p>
            </div>
            <div className='rounded-lg border p-3'>
              <div className='font-medium'>Shared</div>
              <p className='text-muted-foreground'>The channel can be eligible for shared traffic when backend support is enabled.</p>
            </div>
            <div className='rounded-lg border p-3'>
              <div className='font-medium'>5h refresh quota</div>
              <p className='text-muted-foreground'>Treat refresh eligibility as limited to roughly once every 5h per shared channel.</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Go to existing pages</CardTitle>
          <CardDescription>Use the established management screens for the actual channel and model work.</CardDescription>
        </CardHeader>
        <CardContent className='grid gap-3 md:grid-cols-3'>
          {guideLinks.map((link) => (
            <div key={link.title} className='rounded-lg border p-4'>
              <div className='font-medium'>{link.title}</div>
              <p className='text-muted-foreground mt-1 min-h-10 text-sm'>{link.description}</p>
              <Button asChild variant='outline' className='mt-4 w-full'>
                <Link to={link.to}>Open</Link>
              </Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

export const Route = createFileRoute('/_authenticated/share')({
  component: SharePage,
});
