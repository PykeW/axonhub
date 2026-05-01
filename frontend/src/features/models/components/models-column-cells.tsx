import { useCallback, useState } from 'react';
import type { Row } from '@tanstack/react-table';
import { IconLink } from '@tabler/icons-react';
import { usePermissions } from '@/hooks/usePermissions';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { useModels } from '../context/use-models-context';
import type { Model } from '../data/schema';
import { ModelsStatusDialog } from './models-status-dialog';
import { useDeveloperLabel } from './use-developer-label';

// Status Switch Cell Component to handle status toggle with confirmation dialog
export function StatusSwitchCell({ row }: { row: Row<Model> }) {
  const model = row.original;
  const [dialogOpen, setDialogOpen] = useState(false);

  const isEnabled = model.status === 'enabled';
  const isArchived = model.status === 'archived';

  const handleSwitchClick = useCallback(() => {
    if (!isArchived) {
      setDialogOpen(true);
    }
  }, [isArchived]);

  return (
    <>
      <Switch checked={isEnabled} onCheckedChange={handleSwitchClick} disabled={isArchived} data-testid='model-status-switch' />
      {dialogOpen && <ModelsStatusDialog open={dialogOpen} onOpenChange={setDialogOpen} currentRow={model} />}
    </>
  );
}

// Developer Cell Component to show translated developer name
export function DeveloperCell({ row }: { row: Row<Model> }) {
  const getDeveloperLabel = useDeveloperLabel();
  return <Badge variant='outline'>{getDeveloperLabel(row.getValue('developer'))}</Badge>;
}

// Association Rules Cell Component to handle permission check
export function AssociationRulesCell({ row }: { row: Row<Model> }) {
  const model = row.original;
  const { setOpen, setCurrentRow } = useModels();
  const { channelPermissions } = usePermissions();

  const handleOpenAssociationDialog = useCallback(() => {
    setCurrentRow(model);
    setOpen('association');
  }, [model, setCurrentRow, setOpen]);

  const associationCount = model.settings?.associations?.length || 0;

  // Only show button if user has write permissions
  if (!channelPermissions.canWrite) {
    return (
      <div className='flex justify-center'>
        <Badge variant='secondary'>{associationCount}</Badge>
      </div>
    );
  }

  return (
    <Button size='sm' variant='outline' className='h-8 px-3' onClick={handleOpenAssociationDialog}>
      <IconLink className='mr-1 h-3 w-3' />
      {`${associationCount}`}
    </Button>
  );
}
