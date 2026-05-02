import React from 'react';
import { DotsHorizontalIcon } from '@radix-ui/react-icons';
import { Link } from '@tanstack/react-router';
import { Row } from '@tanstack/react-table';
import { IconArchive, IconEdit, IconSettings, IconUserCheck, IconUserOff } from '@tabler/icons-react';
import { BarChart3 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { usePermissions } from '@/hooks/usePermissions';
import { useApiKeysContext } from '../context/use-apikeys-context';
import { ApiKey } from '../data/schema';
import { ApiKeyTokenChartDialog } from './api-key-token-chart-dialog';

interface DataTableRowActionsProps {
  row: Row<ApiKey>;
}

export function DataTableRowActions({ row }: DataTableRowActionsProps) {
  const { t } = useTranslation();
  const { openDialog } = useApiKeysContext();
  const { apiKeyPermissions } = usePermissions();
  const apiKey = row.original;
  const [open, setOpen] = React.useState(false);
  const [chartOpen, setChartOpen] = React.useState(false);

  if (!apiKeyPermissions.canRead && !apiKeyPermissions.canWrite) {
    return null;
  }

  const handleEdit = (target: ApiKey) => {
    setOpen(false);
    setTimeout(() => openDialog('edit', target), 0);
  };

  const handleStatusChange = (target: ApiKey) => {
    if (target.status === 'archived') {
      return;
    }

    setOpen(false);
    setTimeout(() => openDialog('status', target), 0);
  };

  const handleArchive = (target: ApiKey) => {
    setOpen(false);
    setTimeout(() => openDialog('archive', target), 0);
  };

  const handleProfiles = (target: ApiKey) => {
    setOpen(false);
    setTimeout(() => openDialog('profiles', target), 0);
  };

  const handleViewChart = () => {
    setOpen(false);
    setTimeout(() => setChartOpen(true), 0);
  };

  return (
    <>
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger asChild>
          <Button variant='ghost' className='data-[state=open]:bg-muted flex h-8 w-8 p-0'>
            <DotsHorizontalIcon className='h-4 w-4' />
            <span className='sr-only'>Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end' className='w-[190px]'>
          <DropdownMenuItem onClick={handleViewChart}>
            <BarChart3 className='mr-2 h-4 w-4' />
            {t('apikeys.actions.viewTokenChart')}
          </DropdownMenuItem>
          {apiKeyPermissions.canWrite && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => handleEdit(apiKey)}>
                <IconEdit className='mr-2 h-4 w-4' />
                {t('common.actions.edit')}
              </DropdownMenuItem>
              {apiKey.type === 'user' ? (
                <DropdownMenuItem asChild>
                  <Link to='/use' search={{ apiKeyId: apiKey.id }} onClick={() => setOpen(false)}>
                    <IconSettings className='mr-2 h-4 w-4' />
                    {t('apikeys.actions.useSetup')}
                  </Link>
                </DropdownMenuItem>
              ) : null}
              {apiKey.type !== 'service_account' ? (
                <DropdownMenuItem onClick={() => handleProfiles(apiKey)}>
                  <IconSettings className='mr-2 h-4 w-4' />
                  {t('apikeys.actions.profiles')}
                </DropdownMenuItem>
              ) : null}
              {apiKey.status !== 'archived' ? (
                <DropdownMenuItem
                  onClick={() => handleStatusChange(apiKey)}
                  className={apiKey.status === 'enabled' ? 'text-orange-600' : 'text-green-600'}
                >
                  {apiKey.status === 'enabled' ? (
                    <>
                      <IconUserOff className='mr-2 h-4 w-4' />
                      {t('common.buttons.disable')}
                    </>
                  ) : (
                    <>
                      <IconUserCheck className='mr-2 h-4 w-4' />
                      {t('common.buttons.enable')}
                    </>
                  )}
                </DropdownMenuItem>
              ) : null}
              {apiKey.status !== 'archived' ? (
                <DropdownMenuItem onClick={() => handleArchive(apiKey)} className='text-orange-600'>
                  <IconArchive className='mr-2 h-4 w-4' />
                  {t('common.buttons.archive')}
                </DropdownMenuItem>
              ) : null}
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      <ApiKeyTokenChartDialog apiKey={apiKey} open={chartOpen} onOpenChange={setChartOpen} />
    </>
  );
}
