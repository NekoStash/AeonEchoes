export { buildLineDiff } from './diff'
export type { DiffLine, DiffLineKind } from './diff'
export {
  draftDiffersFromBackend,
  EDITOR_DRAFT_SCHEMA_VERSION,
  EDITOR_DRAFT_STORAGE_PREFIX,
  editorDraftStorageKey,
  readEditorDraft,
  removeEditorDraft,
  writeEditorDraft
} from './storage'
export type { DraftStorage, DraftStorageResult, EditorDraftSnapshot } from './storage'
