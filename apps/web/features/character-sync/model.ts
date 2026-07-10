import type { CharacterSyncResponse, StoryBible } from '~/entities/story-bible'

export type CharacterSyncState = 'idle' | 'syncing' | 'synced' | 'failed'

export function countSyncableCharacters(bible: StoryBible): number {
  return bible.characters.filter((character) => (
    character.name.trim()
    && character.role.trim()
    && character.desire.trim()
    && character.wound.trim()
  )).length
}

export function applyCharacterSyncResult(bible: StoryBible, response: CharacterSyncResponse): StoryBible {
  const entitiesById = new Map(response.characters.map((entity) => [entity.id, entity]))
  const mappingsByName = new Map(response.mappings.map((mapping) => [mapping.name.trim(), mapping]))
  return {
    ...bible,
    characters: bible.characters.map((character) => {
      const mapping = mappingsByName.get(character.name.trim())
      if (!mapping) return character
      const entity = entitiesById.get(mapping.entity_id)
      return {
        ...character,
        entity_id: mapping.entity_id,
        sync_status: mapping.action,
        synced_at: entity?.updated_at,
        summary: character.summary || entity?.summary
      }
    })
  }
}
