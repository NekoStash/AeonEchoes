import type { ProjectSummary } from '~/lib/types'

export type ProjectSortKey = 'updated' | 'created' | 'title'
export type StoryBibleFilter = 'all' | ProjectSummary['bible_status']
export type RecentProjectFilter = 'all' | 'recent' | 'other'

export interface ProjectLibraryFilters {
  query: string
  status: string
  storyBible: StoryBibleFilter
  recent: RecentProjectFilter
  sort: ProjectSortKey
}

export function createProjectLibraryFilters(): ProjectLibraryFilters {
  return {
    query: '',
    status: '',
    storyBible: 'all',
    recent: 'all',
    sort: 'updated'
  }
}

export function projectChapterCount(project: ProjectSummary): number | null {
  return typeof project.chapter_count === 'number' && Number.isFinite(project.chapter_count)
    ? project.chapter_count
    : null
}

export function projectSearchText(project: ProjectSummary) {
  const seed = project.seed
  return [
    project.title,
    project.id,
    project.slug,
    project.status,
    project.logline,
    seed?.premise,
    seed?.genre,
    seed?.tone,
    seed?.audience,
    project.active_story_bible_id,
    ...project.tags
  ].filter(Boolean).join('\n').toLocaleLowerCase()
}

export function filterProjects(
  projects: ProjectSummary[],
  filters: ProjectLibraryFilters,
  recentProjectIds: ReadonlySet<string>
) {
  const terms = filters.query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  const matching = projects.filter((project) => {
    if (filters.status && project.status !== filters.status) return false
    if (filters.storyBible !== 'all' && project.bible_status !== filters.storyBible) return false
    if (filters.recent === 'recent' && !recentProjectIds.has(project.id)) return false
    if (filters.recent === 'other' && recentProjectIds.has(project.id)) return false
    if (terms.length === 0) return true
    const corpus = projectSearchText(project)
    return terms.every((term) => corpus.includes(term))
  })

  return matching.sort((left, right) => {
    if (filters.sort === 'title') return left.title.localeCompare(right.title)
    const key = filters.sort === 'created' ? 'created_at' : 'updated_at'
    return timestamp(right[key]) - timestamp(left[key])
  })
}

function timestamp(value?: string) {
  if (!value) return 0
  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : 0
}
