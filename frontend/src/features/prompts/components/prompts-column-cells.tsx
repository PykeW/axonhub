import { useCallback, useState } from 'react';
import type { Row } from '@tanstack/react-table';
import { Switch } from '@/components/ui/switch';
import type { Prompt } from '../data/schema';
import { PromptsStatusDialog } from './prompts-status-dialog';

export function StatusSwitchCell({ row }: { row: Row<Prompt> }) {
  const prompt = row.original;
  const [dialogOpen, setDialogOpen] = useState(false);

  const isEnabled = prompt.status === 'enabled';

  const handleSwitchClick = useCallback(() => {
    setDialogOpen(true);
  }, []);

  return (
    <>
      <Switch checked={isEnabled} onCheckedChange={handleSwitchClick} data-testid='prompt-status-switch' />
      {dialogOpen && <PromptsStatusDialog open={dialogOpen} onOpenChange={setDialogOpen} currentRow={prompt} />}
    </>
  );
}
