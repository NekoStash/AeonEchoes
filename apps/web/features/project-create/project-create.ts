import type { InitializeProjectResponse, ProjectSeed, ProjectSummary } from '~/lib/types'

export type ProjectCreateStep = 'core' | 'world' | 'voice' | 'review'

export const projectCreateSteps: ProjectCreateStep[] = ['core', 'world', 'voice', 'review']

export function splitProjectTags(value: string) {
  return Array.from(new Set(value.split(/[，,]/).map((tag) => tag.trim()).filter(Boolean)))
}

export function isProjectCreateStepComplete(step: ProjectCreateStep, seed: ProjectSeed): boolean {
  if (step === 'core') return Boolean(seed.title?.trim() && seed.one_sentence_core.trim())
  if (step === 'world') return Boolean(seed.world_background.trim() && seed.protagonist.trim() && seed.central_conflict.trim())
  if (step === 'voice') return Boolean(seed.style.trim() && seed.taboos.trim())
  return projectCreateSteps.slice(0, 3).every((item) => isProjectCreateStepComplete(item, seed))
}

export function projectSummaryFromInitialization(result: InitializeProjectResponse): ProjectSummary {
  return {
    id: result.project.id,
    title: result.project.title,
    slug: result.project.slug,
    status: result.project.status,
    logline: result.story_bible.premise,
    tags: [...result.story_bible.themes],
    seed: result.project.seed,
    active_story_bible_id: result.project.active_story_bible_id || result.story_bible.id,
    created_at: result.project.created_at,
    updated_at: result.project.updated_at,
    bible_status: result.story_bible.approved ? 'ready' : 'draft',
    chapter_count: 0,
    target_chapters: result.project.seed?.target_chapters
  }
}

export function createdProjectDestinations(projectId: string) {
  return {
    storyBible: `/projects/${projectId}?section=story`,
    newChapter: `/projects/${projectId}?createChapter=1`
  }
}
