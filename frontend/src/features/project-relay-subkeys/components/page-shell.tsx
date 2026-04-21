import { Header } from '@/components/layout/header';
import { Main } from '@/components/layout/main';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

type PageActionVariant = 'default' | 'outline' | 'secondary' | 'ghost';

interface PageAction {
  label: string;
  variant?: PageActionVariant;
  disabled?: boolean;
}

interface SummaryItem {
  label: string;
  value: string;
  hint: string;
}

interface PageSection {
  title: string;
  description: string;
  bullets: string[];
  badge?: string;
  footer?: string;
}

interface ProjectRelaySubkeysPageShellProps {
  title: string;
  description: string;
  status?: string;
  actions?: PageAction[];
  summary?: SummaryItem[];
  sections: PageSection[];
  asideTitle: string;
  asideDescription: string;
  asideBullets: string[];
}

export function ProjectRelaySubkeysPageShell({
  title,
  description,
  status = 'Skeleton',
  actions = [],
  summary = [],
  sections,
  asideTitle,
  asideDescription,
  asideBullets,
}: ProjectRelaySubkeysPageShellProps) {
  return (
    <>
      <Header fixed>
        <div className='flex flex-1 flex-col gap-4 lg:flex-row lg:items-start lg:justify-between'>
          <div className='space-y-3'>
            <div className='flex flex-wrap items-center gap-2'>
              <Badge>Relay Subkeys</Badge>
              <Badge variant='secondary'>{status}</Badge>
            </div>
            <div>
              <h2 className='text-xl font-bold tracking-tight'>{title}</h2>
              <p className='text-sm text-muted-foreground'>{description}</p>
            </div>
          </div>
          {actions.length > 0 ? (
            <div className='flex flex-wrap gap-2'>
              {actions.map((action) => (
                <Button
                  key={action.label}
                  variant={action.variant ?? 'outline'}
                  disabled={action.disabled ?? true}
                >
                  {action.label}
                </Button>
              ))}
            </div>
          ) : null}
        </div>
      </Header>

      <Main className='space-y-6'>
        {summary.length > 0 ? (
          <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
            {summary.map((item) => (
              <Card key={item.label}>
                <CardHeader>
                  <CardDescription>{item.label}</CardDescription>
                  <CardTitle className='text-2xl'>{item.value}</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className='text-sm text-muted-foreground'>{item.hint}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : null}

        <div className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
          <div className='space-y-6'>
            {sections.map((section) => (
              <Card key={section.title}>
                <CardHeader>
                  <div className='flex flex-wrap items-center gap-2'>
                    <CardTitle>{section.title}</CardTitle>
                    {section.badge ? <Badge variant='outline'>{section.badge}</Badge> : null}
                  </div>
                  <CardDescription>{section.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <ul className='space-y-2 text-sm text-muted-foreground'>
                    {section.bullets.map((bullet) => (
                      <li key={bullet} className='rounded-lg border border-dashed px-3 py-2'>
                        {bullet}
                      </li>
                    ))}
                  </ul>
                </CardContent>
                {section.footer ? (
                  <CardFooter>
                    <p className='text-sm text-muted-foreground'>{section.footer}</p>
                  </CardFooter>
                ) : null}
              </Card>
            ))}
          </div>

          <Card className='h-fit'>
            <CardHeader>
              <CardTitle>{asideTitle}</CardTitle>
              <CardDescription>{asideDescription}</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className='space-y-2 text-sm text-muted-foreground'>
                {asideBullets.map((bullet) => (
                  <li key={bullet} className='rounded-lg bg-muted/40 px-3 py-2'>
                    {bullet}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>
      </Main>
    </>
  );
}
