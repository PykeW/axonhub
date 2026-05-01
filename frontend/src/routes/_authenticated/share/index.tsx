import { Link, createFileRoute } from '@tanstack/react-router';
import { IconAi, IconArrowRight, IconCheck, IconRoute, IconSettings } from '@tabler/icons-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Main } from '@/components/layout/main';

const workflowSteps = [
  'Create or enable provider Channels and keep health checks passing.',
  'Expose only the supported models you want others to consume.',
  'Use model mapping, pricing, and request logs to verify routing behavior.',
];

function ShareMvpPage() {
  return (
    <Main className='space-y-6'>
      <section className='space-y-3'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant='secondary'>共享</Badge>
          <Badge variant='outline'>/share</Badge>
        </div>
        <div className='max-w-3xl space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>共享 / Share</h1>
          <p className='text-muted-foreground'>
            Publish usable model capacity by configuring existing Channels first. This MVP page keeps the flow simple and links to the stable Channel and Model tools instead of introducing new backend forms.
          </p>
        </div>
      </section>

      <div className='grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]'>
        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <IconAi className='h-5 w-5' />
              Share workflow
            </CardTitle>
            <CardDescription>Prepare channels so other API keys can route to shared models safely.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <ol className='space-y-3'>
              {workflowSteps.map((step, index) => (
                <li key={step} className='flex gap-3 rounded-lg border p-3'>
                  <span className='flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground'>
                    {index + 1}
                  </span>
                  <span className='text-sm text-muted-foreground'>{step}</span>
                </li>
              ))}
            </ol>
            <div className='flex flex-wrap gap-2'>
              <Button asChild>
                <Link to='/channels'>
                  Open Channels
                  <IconArrowRight className='h-4 w-4' />
                </Link>
              </Button>
              <Button asChild variant='outline'>
                <Link to='/models'>Review Models</Link>
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Channel checklist</CardTitle>
            <CardDescription>Keep this guidance in sync with the existing Channel settings.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3 text-sm text-muted-foreground'>
            <div className='flex gap-2'>
              <IconCheck className='mt-0.5 h-4 w-4 shrink-0 text-green-500' />
              <span>Enable the channel and confirm the test model succeeds.</span>
            </div>
            <div className='flex gap-2'>
              <IconSettings className='mt-0.5 h-4 w-4 shrink-0 text-blue-500' />
              <span>Set supported models, model mappings, and optional rate limits from Channel settings.</span>
            </div>
            <div className='flex gap-2'>
              <IconRoute className='mt-0.5 h-4 w-4 shrink-0 text-purple-500' />
              <span>Use ordering weight and health status to decide which shared channels should be preferred.</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </Main>
  );
}

export const Route = createFileRoute('/_authenticated/share/')({
  component: ShareMvpPage,
});
