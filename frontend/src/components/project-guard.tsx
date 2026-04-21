import { useEffect } from 'react';
import { useRouter } from '@tanstack/react-router';
import { IconFolderOff } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useSelectedProjectId } from '@/stores/projectStore';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { useMyProjects } from '@/features/projects/data/projects';

interface ProjectGuardProps {
  children: React.ReactNode;
  fallbackPath?: string;
  showNoProjectPage?: boolean;
}

export function ProjectGuard({ children, fallbackPath = '/projects', showNoProjectPage = true }: ProjectGuardProps) {
  const router = useRouter();
  const selectedProjectId = useSelectedProjectId();
  const { data: myProjects, isLoading } = useMyProjects();

  const hasSelectedProject = !!selectedProjectId;
  const hasAnyProjects = !isLoading && !!myProjects && myProjects.length > 0;

  useEffect(() => {
    if (!hasSelectedProject && !showNoProjectPage && !isLoading) {
      router.navigate({ to: fallbackPath });
    }
  }, [hasSelectedProject, showNoProjectPage, fallbackPath, router, isLoading]);

  if (isLoading) {
    return null;
  }

  if (!hasSelectedProject) {
    if (showNoProjectPage) {
      return <NoProjectPage hasAnyProjects={hasAnyProjects} />;
    }

    return null;
  }

  return <>{children}</>;
}

function NoProjectPage({ hasAnyProjects }: { hasAnyProjects: boolean }) {
  const { t } = useTranslation();

  return (
    <div className='flex h-screen items-center justify-center'>
      <div className='max-w-md text-center'>
        <div className='mb-6'>
          <IconFolderOff className='mx-auto h-16 w-16 text-orange-500' />
        </div>

        <Alert className='mb-6'>
          <IconFolderOff className='h-4 w-4' />
          <AlertTitle>{t('common.projectGuard.noProjectSelected')}</AlertTitle>
          <AlertDescription>
            {hasAnyProjects ? t('common.projectGuard.pleaseSelectProject') : t('common.projectGuard.pleaseJoinOrCreateProject')}
          </AlertDescription>
        </Alert>
      </div>
    </div>
  );
}
