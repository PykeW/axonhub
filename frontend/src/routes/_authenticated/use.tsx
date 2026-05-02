import { type SubmitEventHandler, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createFileRoute, type FileRoutesByPath, useNavigate } from '@tanstack/react-router';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useApiKey, useApiKeys, useCreateApiKey, useUpdateApiKeyProfiles } from '@/features/apikeys/data/apikeys';
import {
  apiKeyUseStrategySchema,
  type ApiKey,
  type ApiKeyProfile,
  type ApiKeyUseStrategy,
  type UpdateApiKeyProfilesInput,
} from '@/features/apikeys/data/schema';
import { useQueryModels } from '@/gql/models';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import { usePermissions } from '@/hooks/usePermissions';
import { useSelectedProjectId } from '@/stores/projectStore';

const CREATE_NEW_API_KEY_OPTION = '__create_new_api_key__';
const DEFAULT_API_KEY_NAME = 'Use MVP API Key';
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

type SaveSummary = {
  mode: 'create' | 'edit';
  apiKeyName: string;
  profileName: string;
  modelCount: number;
  useStrategy: ApiKeyUseStrategy;
};

type EditableProfileTarget = {
  profile: ApiKeyProfile;
  profileIndex: number;
  activeProfileName: string;
  existsInApiKey: boolean;
};

type UsePageSearch = {
  apiKeyId?: string;
};

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return 'Unknown error';
}

function isAuthorizationIssue(error: unknown) {
  const status =
    typeof error === 'object' && error !== null && 'status' in error ? Number((error as { status?: unknown }).status) : undefined;
  const message = getErrorMessage(error).toLowerCase();

  return status === 401 || status === 403 || message.includes('unauthorized') || message.includes('forbidden') || message.includes('permission');
}

function createDefaultProfile(name: string = DEFAULT_PROFILE_NAME): ApiKeyProfile {
  return {
    name,
    modelMappings: [],
    channelIDs: [],
    channelTags: [],
    channelTagsMatchMode: 'any',
    modelIDs: [],
    loadBalanceStrategy: null,
    useStrategy: DEFAULT_USE_STRATEGY,
    quota: null,
  };
}

function buildProfileInput(modelIDs: string[], useStrategy: ApiKeyUseStrategy): UpdateApiKeyProfilesInput {
  const profile = createDefaultProfile(DEFAULT_PROFILE_NAME);

  return {
    activeProfile: profile.name,
    profiles: [
      {
        ...profile,
        modelIDs,
        useStrategy,
      },
    ],
  };
}

function resolveEditableProfile(apiKey: ApiKey | null | undefined): EditableProfileTarget {
  const profiles = apiKey?.profiles?.profiles ?? [];
  const activeProfileName = apiKey?.profiles?.activeProfile?.trim() ?? '';

  if (activeProfileName) {
    const activeProfileIndex = profiles.findIndex((profile) => profile.name === activeProfileName);
    if (activeProfileIndex >= 0) {
      return {
        profile: profiles[activeProfileIndex],
        profileIndex: activeProfileIndex,
        activeProfileName,
        existsInApiKey: true,
      };
    }
  }

  const defaultProfileIndex = profiles.findIndex((profile) => profile.name === DEFAULT_PROFILE_NAME);
  if (defaultProfileIndex >= 0) {
    return {
      profile: profiles[defaultProfileIndex],
      profileIndex: defaultProfileIndex,
      activeProfileName: profiles[defaultProfileIndex].name,
      existsInApiKey: true,
    };
  }

  if (profiles.length > 0) {
    return {
      profile: profiles[0],
      profileIndex: 0,
      activeProfileName: profiles[0].name,
      existsInApiKey: true,
    };
  }

  const profile = createDefaultProfile(activeProfileName || DEFAULT_PROFILE_NAME);
  return {
    profile,
    profileIndex: -1,
    activeProfileName: profile.name,
    existsInApiKey: false,
  };
}

function buildMergedProfileInput(apiKey: ApiKey, modelIDs: string[], useStrategy: ApiKeyUseStrategy): UpdateApiKeyProfilesInput {
  const editableProfile = resolveEditableProfile(apiKey);
  const existingProfiles = apiKey.profiles?.profiles ?? [];
  const nextProfile: ApiKeyProfile = {
    ...editableProfile.profile,
    modelIDs,
    useStrategy,
  };

  const profiles = editableProfile.existsInApiKey
    ? existingProfiles.map((profile, index) => (index === editableProfile.profileIndex ? nextProfile : profile))
    : [...existingProfiles, nextProfile];

  return {
    activeProfile: editableProfile.activeProfileName || nextProfile.name,
    profiles: profiles.length > 0 ? profiles : [nextProfile],
  };
}

function UsePage() {
  const navigate = useNavigate();
  const search = Route.useSearch() as UsePageSearch;

  const selectedProjectId = useSelectedProjectId();
  const { apiKeyPermissions, modelPermissions } = usePermissions();
  const createApiKey = useCreateApiKey();
  const updateApiKeyProfiles = useUpdateApiKeyProfiles();
  const { data: models, mutateAsync: fetchModels, isPending: isFetchingModels } = useQueryModels();
  const existingApiKeysQuery = useApiKeys(
    {
      first: 100,
      where: {
        typeIn: ['user'],
        statusIn: ['enabled', 'disabled'],
      },
      orderBy: { field: 'CREATED_AT', direction: 'DESC' },
    },
    {
      disableAutoFetch: !selectedProjectId || !apiKeyPermissions.canRead,
    }
  );

  const [apiKeyName, setApiKeyName] = useState(DEFAULT_API_KEY_NAME);
  const [selectedModelIDs, setSelectedModelIDs] = useState<string[]>([]);
  const [useStrategy, setUseStrategy] = useState<string>(DEFAULT_USE_STRATEGY);
  const [modelLoadError, setModelLoadError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [profileSaveError, setProfileSaveError] = useState<string | null>(null);
  const [generatedApiKey, setGeneratedApiKey] = useState<ApiKey | null>(null);
  const [profileSaved, setProfileSaved] = useState(false);
  const [saveSummary, setSaveSummary] = useState<SaveSummary | null>(null);
  const hydratedEditSnapshotRef = useRef('');

  const generatedKey = generatedApiKey?.key ?? '';
  const { isCopied, handleCopy } = useCopyToClipboard({ text: generatedKey });
  const editingApiKeyId = search.apiKeyId?.trim() ?? '';
  const isEditMode = editingApiKeyId.length > 0;
  const editingApiKeyQuery = useApiKey(editingApiKeyId);
  const editingApiKey = editingApiKeyQuery.data;
  const editingProfileTarget = useMemo(() => resolveEditableProfile(editingApiKey), [editingApiKey]);
  const existingApiKeys = useMemo(() => {
    const apiKeys = existingApiKeysQuery.data?.edges.map((edge) => edge.node) ?? [];

    if (editingApiKey && apiKeys.every((apiKey) => apiKey.id !== editingApiKey.id)) {
      return [editingApiKey, ...apiKeys];
    }

    return apiKeys;
  }, [editingApiKey, existingApiKeysQuery.data]);
  const existingApiKeyValue = isEditMode ? editingApiKeyId : CREATE_NEW_API_KEY_OPTION;
  const isLoadingExistingApiKeys = existingApiKeysQuery.isLoading;
  const isLoadingEditingApiKey = isEditMode && (editingApiKeyQuery.isLoading || editingApiKeyQuery.isFetching);
  const editingApiKeyLoadError =
    isEditMode && editingApiKeyQuery.error ? `Could not load the selected API key: ${getErrorMessage(editingApiKeyQuery.error)}` : null;

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

  useEffect(() => {
    hydratedEditSnapshotRef.current = '';
    setSubmitError(null);
    setProfileSaveError(null);
    setGeneratedApiKey(null);
    setProfileSaved(false);
    setSaveSummary(null);

    if (!isEditMode) {
      setApiKeyName(DEFAULT_API_KEY_NAME);
      setSelectedModelIDs([]);
      setUseStrategy(DEFAULT_USE_STRATEGY);
    }
  }, [editingApiKeyId, isEditMode]);

  useEffect(() => {
    if (!isEditMode || !editingApiKey) {
      return;
    }

    const snapshot = JSON.stringify({
      id: editingApiKey.id,
      updatedAt: editingApiKey.updatedAt.toISOString(),
      profileIndex: editingProfileTarget.profileIndex,
      profileName: editingProfileTarget.profile.name,
      modelIDs: editingProfileTarget.profile.modelIDs ?? [],
      useStrategy: editingProfileTarget.profile.useStrategy ?? DEFAULT_USE_STRATEGY,
    });

    if (hydratedEditSnapshotRef.current === snapshot) {
      return;
    }

    setApiKeyName(editingApiKey.name);
    setSelectedModelIDs(editingProfileTarget.profile.modelIDs ?? []);
    setUseStrategy(editingProfileTarget.profile.useStrategy ?? DEFAULT_USE_STRATEGY);
    hydratedEditSnapshotRef.current = snapshot;
  }, [editingApiKey, editingProfileTarget, isEditMode]);

  const handleApiKeySelectionChange = useCallback(
    (value: string) => {
      const nextApiKeyId = value === CREATE_NEW_API_KEY_OPTION ? undefined : value;

      hydratedEditSnapshotRef.current = '';
      setSubmitError(null);
      setProfileSaveError(null);
      setGeneratedApiKey(null);
      setProfileSaved(false);
      setSaveSummary(null);

      if (!nextApiKeyId) {
        setApiKeyName(DEFAULT_API_KEY_NAME);
        setSelectedModelIDs([]);
        setUseStrategy(DEFAULT_USE_STRATEGY);
      }

      navigate({
        search: ((prev: UsePageSearch | undefined) => {
          const nextSearch: UsePageSearch = { ...(prev ?? {}) };

          if (nextApiKeyId) {
            nextSearch.apiKeyId = nextApiKeyId;
          } else {
            delete nextSearch.apiKeyId;
          }

          return nextSearch;
        }) as any,
        replace: true,
      } as any);
    },
    [navigate]
  );

  const availableModels = useMemo(() => [...(models ?? [])].sort((a, b) => a.id.localeCompare(b.id)), [models]);
  const availableModelIDs = useMemo(() => new Set(availableModels.map((model) => model.id)), [availableModels]);
  const unavailableSelectedModelIDs = selectedModelIDs.filter((modelID) => !availableModelIDs.has(modelID));
  const isSaving = createApiKey.isPending || updateApiKeyProfiles.isPending;
  const activeProfileName = isEditMode ? editingProfileTarget.profile.name : DEFAULT_PROFILE_NAME;

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
      return {
        ok: false,
        error: isEditMode
          ? 'Select at least one available model before saving the API key profile.'
          : 'Select at least one available model before creating the API key.',
      };
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
    setSaveSummary(null);

    if (!selectedProjectId) {
      setSubmitError('Select a project before creating or updating a Use API key.');
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

    if (isEditMode) {
      if (!apiKeyPermissions.canEdit) {
        setSubmitError('Authorization issue: write_api_keys permission is required to update an API key.');
        return;
      }

      if (isLoadingEditingApiKey) {
        setSubmitError('Wait for the selected API key to finish loading before saving changes.');
        return;
      }

      if (!editingApiKey) {
        setSubmitError('Select an existing API key before saving model IDs and use strategy.');
        return;
      }

      setGeneratedApiKey(null);

      try {
        await updateApiKeyProfiles.mutateAsync({
          id: editingApiKey.id,
          input: buildMergedProfileInput(editingApiKey, validation.modelIDs, validation.useStrategy),
        });
        setProfileSaved(true);
        setSaveSummary({
          mode: 'edit',
          apiKeyName: editingApiKey.name,
          profileName: editingProfileTarget.profile.name,
          modelCount: validation.modelIDs.length,
          useStrategy: validation.useStrategy,
        });
      } catch (error) {
        const message = isAuthorizationIssue(error)
          ? `Authorization issue while saving API key profile: ${getErrorMessage(error)}`
          : `Could not update model IDs/use strategy: ${getErrorMessage(error)}`;
        setSubmitError(message);
      }
      return;
    }

    if (!apiKeyPermissions.canCreate) {
      setSubmitError('Authorization issue: write_api_keys permission is required to create an API key.');
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
        setSaveSummary({
          mode: 'create',
          apiKeyName: createdApiKey.name,
          profileName: DEFAULT_PROFILE_NAME,
          modelCount: validation.modelIDs.length,
          useStrategy: validation.useStrategy,
        });
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
          <h1 className='text-3xl font-bold tracking-tight'>Create or update a Use API key</h1>
          <p className='text-muted-foreground max-w-3xl text-sm md:text-base'>
            Create a new Use API key, or load an existing key and update the active profile model IDs and useStrategy without touching the rest of its profile fields.
          </p>
        </div>
      </div>

      <div className='grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]'>
        <Card>
          <CardHeader>
            <CardTitle>API key setup</CardTitle>
            <CardDescription>
              Create mode still produces a fresh key. Edit mode only updates the selected API key profile with the current model IDs and useStrategy.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form className='space-y-6' onSubmit={handleSubmit}>
              {!selectedProjectId ? (
                <Alert>
                  <AlertTitle>Select a project first</AlertTitle>
                  <AlertDescription>The Use MVP flow always runs inside the currently selected project context.</AlertDescription>
                </Alert>
              ) : null}

              {submitError ? (
                <Alert>
                  <AlertTitle>{isEditMode ? 'Cannot update Use API key' : 'Cannot create Use API key'}</AlertTitle>
                  <AlertDescription>{submitError}</AlertDescription>
                </Alert>
              ) : null}

              {modelLoadError ? (
                <Alert>
                  <AlertTitle>Model list unavailable</AlertTitle>
                  <AlertDescription>{modelLoadError}</AlertDescription>
                </Alert>
              ) : null}

              {editingApiKeyLoadError ? (
                <Alert>
                  <AlertTitle>Existing API key unavailable</AlertTitle>
                  <AlertDescription>{editingApiKeyLoadError}</AlertDescription>
                </Alert>
              ) : null}

              <div className='space-y-2'>
                <Label htmlFor='use-existing-api-key'>Mode</Label>
                <Select
                  value={existingApiKeyValue}
                  onValueChange={handleApiKeySelectionChange}
                  disabled={isSaving || !selectedProjectId || (!apiKeyPermissions.canRead && !isEditMode)}
                >
                  <SelectTrigger id='use-existing-api-key' className='w-full'>
                    <SelectValue placeholder='Create a new API key' />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={CREATE_NEW_API_KEY_OPTION}>Create a new API key</SelectItem>
                    {existingApiKeys.map((apiKey) => (
                      <SelectItem key={apiKey.id} value={apiKey.id}>
                        {apiKey.name} ({apiKey.status})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className='text-muted-foreground text-sm'>
                  {apiKeyPermissions.canRead
                    ? 'Choose an existing key to update its active profile, or keep create mode for a new key.'
                    : 'Loading existing keys requires read_api_keys permission. Create mode still works with write_api_keys only.'}
                </p>
                {isLoadingExistingApiKeys ? <Skeleton className='h-9 w-full rounded-md' /> : null}
                {apiKeyPermissions.canRead && !isLoadingExistingApiKeys && existingApiKeys.length === 0 ? (
                  <div className='rounded-md border border-dashed p-3 text-sm text-muted-foreground'>No existing user API keys were found in this project yet.</div>
                ) : null}
              </div>

              {isEditMode ? (
                <div className='rounded-md border bg-muted/30 p-3 text-sm'>
                  {isLoadingEditingApiKey ? (
                    <div className='space-y-2'>
                      <Skeleton className='h-4 w-2/3 rounded' />
                      <Skeleton className='h-4 w-1/2 rounded' />
                    </div>
                  ) : editingApiKey ? (
                    <div className='space-y-1'>
                      <div>
                        Editing existing key: <span className='font-medium'>{editingApiKey.name}</span>
                      </div>
                      <div>
                        Active profile in scope: <span className='font-medium'>{editingProfileTarget.profile.name}</span>
                      </div>
                      <div className='text-muted-foreground'>Only model IDs and useStrategy are updated here. Other profile fields stay unchanged.</div>
                    </div>
                  ) : (
                    <div className='text-muted-foreground'>Select an existing API key to load its current profile settings.</div>
                  )}
                </div>
              ) : null}

              <div className='space-y-2'>
                <Label htmlFor='use-api-key-name'>API key name</Label>
                <Input
                  id='use-api-key-name'
                  value={apiKeyName}
                  onChange={(event) => setApiKeyName(event.target.value)}
                  placeholder={DEFAULT_API_KEY_NAME}
                  disabled={isSaving || isEditMode}
                />
                {isEditMode ? <p className='text-muted-foreground text-sm'>Name is shown for context only here. Rename existing keys from the API Keys management page if needed.</p> : null}
              </div>

              <div className='space-y-2'>
                <Label>Use strategy</Label>
                <Select value={useStrategy || DEFAULT_USE_STRATEGY} onValueChange={setUseStrategy} disabled={isSaving || isLoadingEditingApiKey}>
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
                    <p className='text-muted-foreground text-sm'>Select one or more enabled models for the profile currently in scope.</p>
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
                          <Checkbox checked={checked} onCheckedChange={(nextChecked) => toggleModel(model.id, nextChecked === true)} disabled={isSaving || isLoadingEditingApiKey} />
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

              <Button type='submit' disabled={isSaving || !selectedProjectId || isLoadingEditingApiKey}>
                {isSaving ? (isEditMode ? 'Updating Use API key...' : 'Saving Use API key...') : isEditMode ? 'Update Use API key' : 'Create Use API key'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Current scope</CardTitle>
            <CardDescription>Keep this iteration focused on modelIDs and useStrategy while preserving the rest of the existing profile payload.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3 text-sm'>
            <div className='rounded-lg bg-muted/40 p-3'>Mode: {isEditMode ? 'Edit existing API key' : 'Create new API key'}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Selected project: {selectedProjectId ?? 'None'}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Selected models: {selectedModelIDs.length}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Active profile name: {activeProfileName}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Saved strategy default: {useStrategy || DEFAULT_USE_STRATEGY}</div>
            <div className='rounded-lg bg-muted/40 p-3'>Next step after this edit flow: surface richer profile options and wire Share settings through schema, API, and UI.</div>
          </CardContent>
        </Card>
      </div>

      {saveSummary?.mode === 'edit' ? (
        <Alert>
          <AlertTitle>Use API key updated</AlertTitle>
          <AlertDescription className='space-y-2'>
            <p className='text-sm'>Saved model IDs and useStrategy for {saveSummary.apiKeyName}.</p>
            <p className='text-sm text-muted-foreground'>Profile {saveSummary.profileName} now tracks {saveSummary.modelCount} model IDs and strategy {saveSummary.useStrategy}.</p>
          </AlertDescription>
        </Alert>
      ) : null}

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
            {saveSummary?.mode === 'create' ? <p className='text-sm text-muted-foreground'>Profile {saveSummary.profileName} was saved with {saveSummary.modelCount} model IDs and strategy {saveSummary.useStrategy}.</p> : null}
            {profileSaveError ? <p className='text-sm text-destructive'>{profileSaveError}</p> : null}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

export const Route = createFileRoute('/_authenticated/use' as keyof FileRoutesByPath)({
  component: UsePage,
  validateSearch: (search: UsePageSearch) => ({
    apiKeyId: typeof search.apiKeyId === 'string' && search.apiKeyId.trim().length > 0 ? search.apiKeyId : undefined,
  }),
});
