import { defineStore } from 'pinia'
import { ApiClientError } from '~/lib/api'
import type {
  ApiErrorState,
  GraphExpandResponse,
  HealthStatus,
  IndexJob,
  ModelConfig,
  ProjectSummary,
  ProviderConfig,
  StoryBible
} from '~/lib/types'

const OPENED_PROJECTS_STORAGE_KEY = 'aeon-echoes:opened-projects'

interface WorkspaceState {
  health: HealthStatus | null
  providers: ProviderConfig[]
  models: ModelConfig[]
  projects: ProjectSummary[]
  openedProjects: ProjectSummary[]
  activeBible: StoryBible | null
  activeGraph: GraphExpandResponse | null
  indexJobs: IndexJob[]
  errors: ApiErrorState[]
  loading: Record<string, boolean>
}

function isProjectSummary(value: unknown): value is ProjectSummary {
  if (!value || typeof value !== 'object') return false
  const project = value as Partial<ProjectSummary>
  return typeof project.id === 'string'
    && typeof project.title === 'string'
    && typeof project.logline === 'string'
    && Array.isArray(project.tags)
    && project.tags.every((tag) => typeof tag === 'string')
    && typeof project.updated_at === 'string'
    && (project.bible_status === 'missing' || project.bible_status === 'draft' || project.bible_status === 'ready')
    && typeof project.chapter_count === 'number'
}

function createErrorState(scope: string, error: unknown): ApiErrorState {
  if (error instanceof ApiClientError) return error.state
  if (error instanceof Error) {
    return {
      endpoint: scope,
      message: error.message,
      cause: error
    }
  }
  return {
    endpoint: scope,
    message: `API request failed for ${scope}`,
    cause: error
  }
}

export const useWorkspaceStore = defineStore('workspace', {
  state: (): WorkspaceState => ({
    health: null,
    providers: [],
    models: [],
    projects: [],
    openedProjects: [],
    activeBible: null,
    activeGraph: null,
    indexJobs: [],
    errors: [],
    loading: {}
  }),
  getters: {
    enabledProviders: (state) => state.providers.filter((provider) => provider.enabled),
    enabledModels: (state) => state.models.filter((model) => model.enabled),
    hasApiErrors: (state) => state.errors.length > 0
  },
  actions: {
    setLoading(key: string, value: boolean) {
      this.loading[key] = value
    },
    recordResult<T extends { error?: ApiErrorState }>(scope: string, result: T) {
      if (result.error) {
        console.error(`[AeonEchoes API] ${scope} returned an error`, result.error)
        this.errors = [result.error, ...this.errors].slice(0, 8)
      }
    },
    recordError(scope: string, error: unknown) {
      const state = createErrorState(scope, error)
      console.error(`[AeonEchoes API] ${scope} failed`, state)
      this.errors = [state, ...this.errors].slice(0, 8)
      return state
    },
    clearErrors() {
      this.errors = []
    },
    openProject(project: ProjectSummary) {
      this.openedProjects = [project, ...this.openedProjects.filter((item) => item.id !== project.id)]
      this.persistOpenedProjects()
    },
    closeProject(projectId: string) {
      this.openedProjects = this.openedProjects.filter((project) => project.id !== projectId)
      this.persistOpenedProjects()
    },
    isProjectOpen(projectId: string) {
      return this.openedProjects.some((project) => project.id === projectId)
    },
    hydrateOpenedProjects() {
      if (!import.meta.client) return

      const rawProjects = localStorage.getItem(OPENED_PROJECTS_STORAGE_KEY)
      if (!rawProjects) {
        this.openedProjects = []
        return
      }

      try {
        const parsed = JSON.parse(rawProjects)
        if (!Array.isArray(parsed)) {
          throw new Error('Opened projects storage payload is not an array')
        }
        this.openedProjects = parsed.filter(isProjectSummary)
      } catch (error) {
        console.error('Failed to hydrate opened projects from localStorage', error)
        this.openedProjects = []
      }
    },
    persistOpenedProjects() {
      if (!import.meta.client) return

      try {
        localStorage.setItem(OPENED_PROJECTS_STORAGE_KEY, JSON.stringify(this.openedProjects))
      } catch (error) {
        console.error('Failed to persist opened projects to localStorage', error)
      }
    },
    syncOpenedProjectsWithLoadedProjects() {
      if (this.openedProjects.length === 0 || this.projects.length === 0) return

      const latestProjects = new Map(this.projects.map((project) => [project.id, project]))
      this.openedProjects = this.openedProjects.map((project) => latestProjects.get(project.id) || project)
      this.persistOpenedProjects()
    },
    async loadDashboard() {
      const api = useApi()
      this.setLoading('dashboard', true)
      try {
        const [health, projects, providers, models] = await Promise.allSettled([
          api.health(),
          api.listProjects(),
          api.listProviders(),
          api.listModels()
        ])

        if (health.status === 'fulfilled') {
          this.recordResult('health', health.value)
          this.health = health.value.data
        } else {
          this.recordError('health', health.reason)
        }

        if (projects.status === 'fulfilled') {
          this.recordResult('projects', projects.value)
          this.projects = projects.value.data
        } else {
          this.recordError('projects', projects.reason)
        }

        if (providers.status === 'fulfilled') {
          this.recordResult('providers', providers.value)
          this.providers = providers.value.data
        } else {
          this.recordError('providers', providers.reason)
        }

        if (models.status === 'fulfilled') {
          this.recordResult('models', models.value)
          this.models = models.value.data
        } else {
          this.recordError('models', models.reason)
        }

        this.hydrateOpenedProjects()
        this.syncOpenedProjectsWithLoadedProjects()
      } finally {
        this.setLoading('dashboard', false)
      }
    },
    async loadProvidersAndModels(kind?: string) {
      const api = useApi()
      this.setLoading('providers', true)
      try {
        const [providers, models] = await Promise.allSettled([api.listProviders(), api.listModels(kind)])

        if (providers.status === 'fulfilled') {
          this.recordResult('providers', providers.value)
          this.providers = providers.value.data
        } else {
          this.recordError('providers', providers.reason)
        }

        if (models.status === 'fulfilled') {
          this.recordResult('models', models.value)
          this.models = models.value.data
        } else {
          this.recordError('models', models.reason)
        }
      } finally {
        this.setLoading('providers', false)
      }
    },
    async refreshModels(providerId: string) {
      const api = useApi()
      this.setLoading(`models:${providerId}`, true)
      try {
        const result = await api.refreshModels(providerId)
        this.recordResult('model-refresh', result)
        const providerModelIds = new Set(result.data.map((model) => model.id))
        this.models = [...this.models.filter((model) => !providerModelIds.has(model.id)), ...result.data]
        const providers = await api.listProviders()
        this.recordResult('providers', providers)
        this.providers = providers.data
        return result
      } catch (error) {
        this.recordError('model-refresh', error)
        throw error
      } finally {
        this.setLoading(`models:${providerId}`, false)
      }
    },
    async loadStoryBible(projectId: string) {
      const api = useApi()
      this.setLoading(`bible:${projectId}`, true)
      try {
        const result = await api.getStoryBible(projectId)
        this.recordResult('story-bible', result)
        this.activeBible = result.data
        return result
      } catch (error) {
        this.recordError('story-bible', error)
        throw error
      } finally {
        this.setLoading(`bible:${projectId}`, false)
      }
    },
    async updateStoryBible(projectId: string, bible: StoryBible) {
      const api = useApi()
      this.setLoading(`bible-save:${projectId}`, true)
      try {
        const result = await api.updateStoryBible(projectId, bible)
        this.recordResult('story-bible-save', result)
        this.activeBible = result.data
        return result
      } catch (error) {
        this.recordError('story-bible-save', error)
        throw error
      } finally {
        this.setLoading(`bible-save:${projectId}`, false)
      }
    },
    async syncCharacters(projectId: string, bible: StoryBible) {
      const api = useApi()
      this.setLoading(`characters-sync:${projectId}`, true)
      try {
        const result = await api.syncCharacters(projectId, bible)
        this.recordResult('characters-sync', result)
        if (this.activeBible?.project_id === projectId) {
          const entitiesById = new Map(result.data.characters.map((entity) => [entity.id, entity]))
          const mappingsByName = new Map(result.data.mappings.map((mapping) => [mapping.name.trim(), mapping]))
          this.activeBible = {
            ...this.activeBible,
            characters: this.activeBible.characters.map((character) => {
              const mapping = mappingsByName.get(character.name.trim())
              if (!mapping) return character
              const entity = entitiesById.get(mapping.entity_id)
              return {
                ...character,
                entity_id: mapping.entity_id,
                sync_status: mapping.action || 'synced',
                synced_at: entity?.updated_at,
                summary: character.summary || entity?.summary
              }
            })
          }
        }
        return result
      } catch (error) {
        this.recordError('characters-sync', error)
        throw error
      } finally {
        this.setLoading(`characters-sync:${projectId}`, false)
      }
    },
    async loadGraph(projectId: string, root: string, depth: number, timeline: number, filters: string[]) {
      const api = useApi()
      this.setLoading(`graph:${projectId}`, true)
      try {
        const entityIds = root && root !== 'story_start' ? [root] : undefined
        const result = await api.expandGraph({ project_id: projectId, root, depth, timeline, filters, entity_ids: entityIds })
        this.recordResult('graph', result)
        this.activeGraph = result.data
        return result
      } catch (error) {
        this.recordError('graph', error)
        throw error
      } finally {
        this.setLoading(`graph:${projectId}`, false)
      }
    },
    async loadIndexJobs(projectId?: string) {
      const api = useApi()
      const key = `index-jobs:${projectId || 'all'}`
      this.setLoading(key, true)
      try {
        const result = await api.listIndexJobs(projectId)
        this.recordResult('index-jobs', result)
        this.indexJobs = result.data
        return result
      } catch (error) {
        this.recordError('index-jobs', error)
        return null
      } finally {
        this.setLoading(key, false)
      }
    },
    async runPendingIndexJobs(projectId?: string, limit = 10) {
      const api = useApi()
      const key = `index-run-pending:${projectId || 'all'}`
      this.setLoading(key, true)
      try {
        const result = await api.runPendingIndexJobs(projectId, limit)
        this.recordResult('index-run-pending', result)
        if (result.data.error) {
          this.recordError('index-run-pending', new Error(result.data.error))
        }
        if (result.data.processed.length > 0) {
          const updatedIds = new Set(result.data.processed.map((job) => job.id))
          this.indexJobs = [
            ...result.data.processed,
            ...this.indexJobs.filter((job) => !updatedIds.has(job.id))
          ]
        }
        return result
      } catch (error) {
        this.recordError('index-run-pending', error)
        throw error
      } finally {
        this.setLoading(key, false)
      }
    }
  }
})
