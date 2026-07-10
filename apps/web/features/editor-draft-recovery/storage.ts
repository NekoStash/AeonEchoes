export const EDITOR_DRAFT_SCHEMA_VERSION = 2 as const
export const EDITOR_DRAFT_STORAGE_PREFIX = 'aeon-echoes:chapter-draft'

export interface EditorDraftSnapshot {
  schema_version: typeof EDITOR_DRAFT_SCHEMA_VERSION
  project_id: string
  chapter_id: string
  title: string
  content: string
  parent_version_id?: string
  updated_at: string
}

export interface DraftStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

export interface DraftStorageResult<T> {
  value: T
  error: Error | null
}

export function editorDraftStorageKey(projectId: string, chapterId: string) {
  const normalizedProjectId = projectId.trim()
  const normalizedChapterId = chapterId.trim()
  if (!normalizedProjectId || !normalizedChapterId) {
    throw new Error('Draft storage requires a real project ID and chapter ID.')
  }
  return `${EDITOR_DRAFT_STORAGE_PREFIX}:v${EDITOR_DRAFT_SCHEMA_VERSION}:${normalizedProjectId}:${normalizedChapterId}`
}

function parseDraft(value: unknown, projectId: string, chapterId: string): EditorDraftSnapshot {
  if (!value || typeof value !== 'object') throw new Error('Stored editor draft is not an object.')
  const draft = value as Partial<EditorDraftSnapshot>
  if (draft.schema_version !== EDITOR_DRAFT_SCHEMA_VERSION) throw new Error('Stored editor draft schema version is unsupported.')
  if (draft.project_id !== projectId || draft.chapter_id !== chapterId) throw new Error('Stored editor draft belongs to another chapter.')
  if (typeof draft.title !== 'string' || typeof draft.content !== 'string' || typeof draft.updated_at !== 'string') {
    throw new Error('Stored editor draft is incomplete.')
  }
  return draft as EditorDraftSnapshot
}

export function readEditorDraft(storage: DraftStorage, projectId: string, chapterId: string): DraftStorageResult<EditorDraftSnapshot | null> {
  const key = editorDraftStorageKey(projectId, chapterId)
  try {
    const raw = storage.getItem(key)
    if (!raw) return { value: null, error: null }
    return { value: parseDraft(JSON.parse(raw), projectId, chapterId), error: null }
  } catch (cause) {
    const error = cause instanceof Error ? cause : new Error('Failed to read the local editor draft.')
    console.error('[AeonEchoes Draft] Failed to read local draft.', error)
    return { value: null, error }
  }
}

export function writeEditorDraft(storage: DraftStorage, snapshot: Omit<EditorDraftSnapshot, 'schema_version' | 'updated_at'> & { updated_at?: string }): DraftStorageResult<EditorDraftSnapshot | null> {
  const draft: EditorDraftSnapshot = {
    ...snapshot,
    schema_version: EDITOR_DRAFT_SCHEMA_VERSION,
    updated_at: snapshot.updated_at || new Date().toISOString()
  }
  const key = editorDraftStorageKey(draft.project_id, draft.chapter_id)
  try {
    storage.setItem(key, JSON.stringify(draft))
    return { value: draft, error: null }
  } catch (cause) {
    const error = cause instanceof Error ? cause : new Error('Failed to write the local editor draft.')
    console.error('[AeonEchoes Draft] Failed to persist local draft.', error)
    return { value: null, error }
  }
}

export function removeEditorDraft(storage: DraftStorage, projectId: string, chapterId: string): DraftStorageResult<boolean> {
  const key = editorDraftStorageKey(projectId, chapterId)
  try {
    storage.removeItem(key)
    return { value: true, error: null }
  } catch (cause) {
    const error = cause instanceof Error ? cause : new Error('Failed to remove the local editor draft.')
    console.error('[AeonEchoes Draft] Failed to remove local draft.', error)
    return { value: false, error }
  }
}

export function draftDiffersFromBackend(draft: EditorDraftSnapshot, title: string, content: string, parentVersionId?: string) {
  return draft.title !== title || draft.content !== content || Boolean(draft.parent_version_id && draft.parent_version_id !== parentVersionId)
}
