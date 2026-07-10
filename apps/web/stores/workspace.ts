import { defineStore } from 'pinia'
import type { ProjectSummary } from '~/entities/project'

const OPENED_PROJECTS_STORAGE_KEY = 'aeon-echoes:opened-projects'

interface WorkspaceState {
  openedProjects: ProjectSummary[]
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
    && (project.chapter_count === undefined || project.chapter_count === null || typeof project.chapter_count === 'number')
}

export const useWorkspaceStore = defineStore('workspace', {
  state: (): WorkspaceState => ({
    openedProjects: []
  }),
  actions: {
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
    syncOpenedProjects(projects: ProjectSummary[]) {
      if (this.openedProjects.length === 0 || projects.length === 0) return

      const latestProjects = new Map(projects.map((project) => [project.id, project]))
      this.openedProjects = this.openedProjects.map((project) => latestProjects.get(project.id) || project)
      this.persistOpenedProjects()
    }
  }
})
