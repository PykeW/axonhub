import { Link, createFileRoute } from '@tanstack/react-router';
import { IconArrowRight, IconKey, IconListDetails, IconRoute, IconSparkles } from '@tabler/icons-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Main } from '@/components/layout/main';

const strategyOptions = [
  {
    title: 'Prefer own channel first',
    description: 'Use project-owned Channels when they support the requested model, then fall back to shared capacity.',
  },
  {
    title: 'Use shared pool first',
    description: 'Route to shared Channels by model availability and health, while keeping owned Channels as a backup.',
  },
  {
    title: 'Explicit model selection',
    description: 'Pin model names in client requests and review logs to confirm which Channel served each request.',
  },
];

function UseMvpPage() {
  return (
    <Main className='space-y-6'>
      <section className='space-y-3'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant='secondary'>使用</Badge>
          <Badge variant='outline'>/use</Badge>
        </div>
        <div className='max-w-3xl space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>使用 / Use</h1>
          <p className='text-muted-foreground'>
            Start consuming shared model capacity with existing Project API Keys. Choose a routing strategy, document desired models, then verify behavior in request logs before deeper backend preferences are enabled.
          </p>
        </div>
      </section>

      <div className='grid gap-4 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]'>
        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <IconKey className='h-5 w-5' />
              Existing entry points
            </CardTitle>
            <CardDescription>Use stable screens for this minimal MVP slice.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            <Button asChild className='w-full justify-between'>
              <Link to='/project/api-keys'>
                Manage Project API Keys
                <IconArrowRight className='h-4 w-4' />
              </Link>
            </Button>
            <Button asChild variant='outline' className='w-full justify-between'>
              <Link to='/models'>
                Review Available Models
                <IconSparkles className='h-4 w-4' />
              </Link>
            </Button>
            <Button asChild variant='outline' className='w-full justify-between'>
              <Link to='/project/requests'>
                Verify Request Logs
                <IconListDetails className='h-4 w-4' />
              </Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <IconRoute className='h-5 w-5' />
              Strategy and model guidance
            </CardTitle>
            <CardDescription>Pick one approach per API key or project until a dedicated preference form is available.</CardDescription>
          </CardHeader>
          <CardContent className='grid gap-3 md:grid-cols-3'>
            {strategyOptions.map((option) => (
              <div key={option.title} className='rounded-lg border p-4'>
                <h2 className='font-semibold'>{option.title}</h2>
                <p className='mt-2 text-sm text-muted-foreground'>{option.description}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </Main>
  );
}

export const Route = createFileRoute('/_authenticated/use/')({
  component: UseMvpPage,
});
