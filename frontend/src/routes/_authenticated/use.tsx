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
    to: '/project/api-keys',
    title: 'Create API Key',
    description: 'Create or manage the key that clients will call with.',
  },
  {
    to: '/models',
    title: 'Choose Model IDs',
    description: 'Confirm the model IDs your key or profile should allow.',
  },
  {
    to: '/project/requests',
    title: 'Check Requests',
    description: 'Review traffic after the key starts using own or shared channels.',
  },
];

const useSteps = [
  'Create an API key from the existing API Keys page.',
  'Choose the model IDs and profile that the key should be allowed to use.',
  'Pick the routing strategy: prefer own, own only, or allow shared.',
  'Use Requests to confirm which traffic succeeds and where follow-up tuning is needed.',
];

const strategies = [
  {
    name: 'Prefer own',
    description: 'Try your own channels first, then fall back to shared channels when allowed.',
  },
  {
    name: 'Own only',
    description: 'Use only channels you own. Shared channels should not be selected.',
  },
  {
    name: 'Allow shared',
    description: 'Permit eligible shared channels for model IDs covered by the selected profile.',
  },
];

function UsePage() {
  return (
    <div className='mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 md:p-8'>
      <div className='space-y-3'>
        <Badge variant='secondary'>Use MVP</Badge>
        <div className='space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>Use available models</h1>
          <p className='text-muted-foreground max-w-3xl text-sm md:text-base'>
            Use this guide page to connect the existing API Keys, Models, and Requests screens. It documents the simplified
            strategy choices without adding a new complex setup form.
          </p>
        </div>
      </div>

      <div className='grid gap-4 md:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>Use workflow</CardTitle>
            <CardDescription>Minimal path for turning selected model IDs into usable API traffic.</CardDescription>
          </CardHeader>
          <CardContent>
            <ol className='list-decimal space-y-3 pl-5 text-sm'>
              {useSteps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Routing strategies</CardTitle>
            <CardDescription>Use these terms when configuring API key/profile behavior.</CardDescription>
          </CardHeader>
          <CardContent className='grid gap-3 text-sm'>
            {strategies.map((strategy) => (
              <div key={strategy.name} className='rounded-lg border p-3'>
                <div className='font-medium'>{strategy.name}</div>
                <p className='text-muted-foreground'>{strategy.description}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Go to existing pages</CardTitle>
          <CardDescription>Use the established pages for API keys, model IDs, profiles, and request review.</CardDescription>
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

export const Route = createFileRoute('/_authenticated/use')({
  component: UsePage,
});
