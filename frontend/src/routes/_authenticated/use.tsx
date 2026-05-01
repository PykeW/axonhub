import { type SubmitEventHandler, useCallback, useEffect, useMemo, useState } from 'react';
import { createFileRoute, type FileRoutesByPath } from '@tanstack/react-router';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useCreateApiKey, useUpdateApiKeyProfiles } from '@/features/apikeys/data/apikeys';
import {
  apiKeyUseStrategySchema,
  type ApiKey,
  type ApiKeyUseStrategy,
  type UpdateApiKeyProfilesInput,
} from '@/features/apikeys/data/schema';
import { useQueryModels } from '@/gql/models';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import { usePermissions } from '@/hooks/usePermissions';
import { useSelectedProjectId } from '@/stores/projectStore';

const DEFAULT_PROFILE_NAME = 'Use MVP';
const DEFAULT_USE_STRATEGY: ApiKeyUseStrategy = 'prefer_own';

const useStrategyOptions: Array<{
  value: ApiKeyUseStrategy;
  title: string;
  description: string;
}> = [
  {
    value: 'prefer_own',
    title: 'Prefer own',
    description: 'Try your own channels first, then fall back to shared channels when eligible.',
  },
  {
    value: 'only_own',
    title: 'Only own',
    description: 'Route only to channels owned by the current project.',
  },
  {
    value: 'allow_shared',
    title: 'Allow shared',
    description: 'Allow eligible shared channels for the selected model IDs.',
  },
];

type ValidateFormResult =
  | {
      ok: false;
      error: string;
    }
  | {
      ok: true;
      name: string;
      modelIDs: string[];
      useStrategy: ApiKeyUseStrategy;
    };

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return 'Unknown error';
}

function isAuthorizationIssue(error: unknown) {
  const status = typeof error === 'object' && error !== null && 'status' in error ? Number((error as { status?: unknown }).status) : undefined;
  const message = getErrorMessage(error).toLowerCase();

  return status === 401 || status === 403 || message.includes('unauthorized') || message.includes('forbidden') || message.includes('permission');
}

function buildProfileInput(modelIDs: string[], useStrategy: ApiKeyUseStrategy): UpdateApiKeyProfilesInput {
  return {
    activeProfile: DEFAULT_PROFILE_NAME,
    profiles: [
      {
        name: DEFAULT_PROFILE_NAME,
        modelMappings: [],
        channelIDs: [],
        channelTags: [],
        channelTagsMatchMode: 'any',
        modelIDs,
        loadBalanceStrategy: null,
        useStrategy,
        quota: null,
      },
    ],
  };
}

function UsePage() {
  const selectedProjectId = useSelectedProjectId();
  const { apiKeyPermissions, modelPermissions } = usePermissions();
  const createApiKey = useCreateApiKey();
  const updateApiKeyProfiles = useUpdateApiKeyProfiles();
  const { data: models, mutateAsync: fetchModels, isPending: isFetchingModels } = useQueryModels();

  const [apiKeyName, setApiKeyName] = useState('Use MVP API Key');
  const [selectedModelIDs, setSelectedModelIDs] = useState<string[]>([]);
  const [useStrategy, setUseStrategy] = useState<string>(DEFAULT_USE_STRATEGY);
  const [modelLoadError, setModelLoadError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [profileSaveError, setProfileSaveError] = useState<string | null>(null);
  const [generatedApiKey, setGeneratedApiKey] = useState<ApiKey | null>(null);
  const [profileSaved, setProfileSaved] = useState(false);

  const generatedKey = generatedApiKey?.key ?? '';
  const { isCopied, handleCopy } = useCopyToClipboard({ text: generatedKey });

  const loadModels = useCallback(async () => {
    setModelLoadError(null);

    try {
      await fetchModels({
        statusIn: ['enabled'],
        includeMapping: true,
        includePrefix: true,
        includeAllChannelModels: true,
      });
    } catch (error) {
      const message = isAuthorizationIssue(error)
        ? `Authorization issue while loading models: ${getErrorMessage(error)}`
        : `Could not load available models: ${getErrorMessage(error)}`;
      setModelLoadError(message);
    }
  }, [fetchModels]);

  useEffect(() => {
    if (selectedProjectId && modelPermissions.canRead) {
      void loadModels();
    }
  }, [loadModels, modelPermissions.canRead, selectedProjectId]);

  const availableModels = useMemo(() => [...(models ?? [])].sort((a, b) => a.id.localeCompare(b.id)), [models]);
  const availableModelIDs = useMemo(() => new Set(availableModels.map((model) => model.id)), [availableModels]);
  const unavailableSelectedModelIDs = selectedModelIDs.filter((modelID) => !availableModelIDs.has(modelID));
  const isSaving = createApiKey.isPending || updateApiKeyProfiles.isPending;

  const toggleModel = (modelID: string, checked: boolean) => {
    setSelectedModelIDs((current) => {
      if (checked) {
        return current.includes(modelID) ? current : [...current, modelID];
      }

      return current.filter((id) => id !== modelID);
    });
  };

  const validateForm = (): ValidateFormResult => {
    const name = apiKeyName.trim();
    if (!name) {
      return { ok: false, error: 'API key name is required.' };
    }

    const modelIDs = selectedModelIDs.filter(Boolean);
    if (modelIDs.length === 0) {
      return { ok: false, error: 'Select at least one available model before creating the API key.' };
    }

    if (unavailableSelectedModelIDs.length > 0) {
      return { ok: false, error: `Selected models are no longer available: ${unavailableSelectedModelIDs.join(', ')}` };
    }

    const parsedUseStrategy = apiKeyUseStrategySchema.safeParse(useStrategy.trim() || DEFAULT_USE_STRATEGY);
    if (!parsedUseStrategy.success) {
      return { ok: false, error: 'Invalid use strategy. Choose prefer_own, only_own, or allow_shared.' };
    }

    return { ok: true, name, modelIDs, useStrategy: parsedUseStrategy.data };
  };

  const handleSubmit: SubmitEventHandler<HTMLFormElement> = async (event) => {
    event.preventDefault();
    setSubmitError(null);
    setProfileSaveError(null);
    setProfileSaved(false);

    if (!selectedProjectId) {
      setSubmitError('Select a project before creating a Use API key.');
      return;
    }

    if (!apiKeyPermissions.canCreate) {
      setSubmitError('Authorization issue: write_api_keys permission is required to create an API key.');
      return;
    }

    if (!modelPermissions.canRead) {
      setSubmitError('Authorization issue: read_channels permission is required to list available models.');
      return;
    }

    const validation = validateForm();
    if (!validation.ok) {
      setSubmitError(validation.error);
      return;
    }

    try {
      const createResult = await createApiKey.mutateAsync({
        name: validation.name,
        type: 'user',
      });
      const createdApiKey = createResult.createAPIKey;
      setGeneratedApiKey(createdApiKey);

      try {
        await updateApiKeyProfiles.mutateAsync({
          id: createdApiKey.id,
          input: buildProfileInput(validation.modelIDs, validation.useStrategy),
        });
        setProfileSaved(true);
      } catch (error) {
        const message = isAuthorizationIssue(error)
          ? `Authorization issue while saving API key profile: ${getErrorMessage(error)}`
          : `API key was created, but saving model IDs/use strategy failed: ${getErrorMessage(error)}`;
        setProfileSaveError(message);
      }
    } catch (error) {
      const message = isAuthorizationIssue(error)
        ? `Authorization issue while creating API key: ${getErrorMessage(error)}`
        : `Could not create API key: ${getErrorMessage(error)}`;
      setSubmitError(message);
    }
  };

  return (
    <div className='mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 md:p-8'>
      <div className='space-y-3'>
        <Badge variant='secondary'>Use MVP</Badge>
        <div className='space-y-2'>
          <h1 className='text-3xl font-bold tracking-tight'>Create a Use API key</h1>
          <p className='text-muted-foreground max-w-3xl text-sm md:text-base'>
            Restore the smallest useful Use flow: create an API key, choose model IDs, save useStrategy, and copy the generated credential.
          </p>
        </div>
      </div>

      <div className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
        <Card>
          <CardHeader>
            <CardTitle>API key setup</CardTitle>
            <CardDescription>Model IDs are required. Empty strategy values are saved as prefer_own.</CardDescription>
          </CardHeader>
          <CardContent>
            <form className='space-y-6' onSubmit={handleSubmit}>
              {!selectedProjectId ? (
                <Alert>
                  <AlertTitle>Select a project first</AlertTitle>
                  <AlertDescription>The Use MVP key is created inside the currently selected project context.</AlertDescription>
                </Alert>
              ) : null}

              {submitError ? (
                <Alert>
                  <AlertTitle>Cannot create Use API key</AlertTitle>
                  <AlertDescription>{submitError}</AlertDescription>
                </Alert>
              ) : null}

              {modelLoadError ? (
                <Alert>
                  <AlertTitle>Model list unavailable</AlertTitle>
                  <AlertDescription>{modelLoadError}</AlertDescription>
                </Alert>
              ) : null}

              <div className='space-y-2'>
                <Label htmlFor='use-api-key-name'>API key name</Label>
                <Input
                  id='use-api-key-name'
                  value={apiKeyName}
                  onChange={(event) => setApiKeyName(event.target.value)}
                  placeholder='Use MVP API Key'
                  disabled={isSaving}
                />
              </div>

              <div className='space-y-2'>
                <Label>Use strategy</Label>
                <Select value={useStrategy || DEFAULT_USE_STRATEGY} onValueChange={setUseStrategy} disabled={isSaving}>
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder='prefer_own' />
                  </SelectTrigger>
                  <SelectContent>
                    {useStrategyOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.value}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <div className='grid gap-2 pt-2'>
                  {useStrategyOptions.map((option) => (
                    <div key={option.value} className='rounded-md border p-3 text-sm'>
                      <div className='font-medium'>
                        {option.title} <span className='text-muted-foreground font-mono text-xs'>({option.value})</span>
                      </div>
                      <p className='text-muted-foreground mt-1'>{option.description}</p>
                    </div>
                  ))}
                </div>
              </div>

              <div className='space-y-3'>
                <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                  <div>
                    <Label>Available models</Label>
                    <p className='text-muted-foreground text-sm'>Select one or more enabled models for this API key profile.</p>
                  </div>
                  <Button type='button' variant='outline' size='sm' onClick={() => void loadModels()} disabled={isFetchingModels || isSaving || !modelPermissions.canRead}>
                    {isFetchingModels ? 'Loading...' : 'Reload models'}
                  </Button>
                </div>

                {isFetchingModels ? (
                  <div className='grid gap-2 sm:grid-cols-2'>
                    {Array.from({ length: 6 }).map((_, index) => (
                      <Skeleton key={index} className='h-16 w-full rounded-md' />
                    ))}
                  </div>
                ) : null}

                {!isFetchingModels && availableModels.length === 0 ? (
                  <div className='rounded-md border border-dashed p-4 text-sm text-muted-foreground'>
                    {selectedProjectId ? 'No enabled models loaded yet. Use Reload models to fetch the current model list.' : 'Pick a project first to load models.'}
                  </div>
                ) : null}

                {!isFetchingModels && availableModels.length > 0 ? (
                  <div className='grid gap-2 sm:grid-cols-2'>
                    {availableModels.map((model) => {
                      const checked = selectedModelIDs.includes(model.id);
                      return (
                        <label key={model.id} className='flex cursor-pointer items-start gap-3 rounded-md border p-3 text-sm'>
                          <Checkbox checked={checked} onCheckedChange={(nextChecked) => toggleModel(model.id, nextChecked === true)} disabled={isSaving} />
                          <div className='space-y-1'>
                            <div className='font-mono text-xs text-foreground'>{model.id}</div>
                            <div className='text-muted-foreground text-xs'>Status: {model.status}</div>
                          </div>
                        </label>
                      );
                    })}
                  </div>
                ) : null}
              </div>

              <Button type='submit' disabled={isSaving || !selectedProjectId}>
                {isSaving ? 'Saving Use API key...' : 'Create Use API key'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Current scope</CardTitle>
            <CardDescription>Keep this first restore narrow and compatible with the committed useStrategy contract.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3 text-sm'>
            <div className='rounded-lg bg-muted/40 p-3'>Selected project: {selectedProjectId ?? 'None'}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Selected models: {selectedModelIDs.length}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Active profile name: {DEFAULT_PROFILE_NAME}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Saved strategy default: {useStrategy || DEFAULT_USE_STRATEGY}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Next step after this page restore: support editing existing Use API keys and saving richer profile options.</div>
          </CardContent>
        </Card>
      </div>

      {generatedApiKey ? (
        <Alert>
          <AlertTitle>{profileSaved ? 'Use API key created' : 'API key created, profile save needs attention'}</AlertTitle>
          <AlertDescription className='space-y-3'>
            <p className='text-sm'>Copy this generated key now. The value may not be shown again in later views.</p>
            <div className='rounded-md border bg-muted/40 p-3 font-mono text-sm break-all'>{generatedKey}</div>
            <div className='flex flex-wrap gap-2'>
              <Button type='button' variant='outline' size='sm' onClick={handleCopy}>
                {isCopied ? 'Copied' : 'Copy API key'}
              </Button>
              <Button type='button' variant='outline' size='sm' onClick={() => void loadModels()} disabled={isFetchingModels}>
                Refresh models
              </Button>
            </div>
            {profileSaved ? <p className='text-sm text-muted-foreground'>Profile {DEFAULT_PROFILE_NAME} was saved with {selectedModelIDs.length} model IDs and strategy {useStrategy || DEFAULT_USE_STRATEGY}.</p> : null}
            {profileSaveError ? <p className='text-sm text-destructive'>{profileSaveError}</p> : null}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
export const Route = createFileRoute('/_authenticated/use' as keyof FileRoutesByPath)({
  component: UsePage,
});
