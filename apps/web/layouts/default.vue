<script setup lang="ts">
import { BookOpenText, Boxes, Bot, Cpu, DatabaseZap, FilePenLine, FolderOpen, Gauge, GitFork, Menu, Plus, Settings2 } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import AppNavigation from '~/widgets/app-navigation/AppNavigation.vue'
import type { AppNavigationGroup } from '~/widgets/app-navigation/navigation'
import { isRouteActive } from '~/widgets/app-navigation/navigation'
import ProjectSwitcher from '~/widgets/project-switcher/ProjectSwitcher.vue'
import { useProjectStore } from '~/entities/project'

const route = useRoute()
const { t } = useI18n()
const workspace = useWorkspaceStore()
const projectStore = useProjectStore()
const { openedProjects } = storeToRefs(workspace)
const { items: projects } = storeToRefs(projectStore)
const mobileMenuOpen = ref(false)

const activeProjectId = computed(() => {
  const matched = route.path.match(/^\/projects\/([^/]+)/)
  if (!matched || matched[1] === 'new') return ''
  return matched[1]
})

const creativeItems = computed(() => [
  { label: t('nav.dashboard'), to: '/', icon: Gauge, exact: true },
  { label: t('nav.projects'), to: '/projects', icon: FolderOpen, exact: true },
  { label: t('nav.newProject'), to: '/projects/new', icon: Plus, exact: true }
])

const projectItems = computed(() => activeProjectId.value ? [
  { label: t('nav.project'), to: `/projects/${activeProjectId.value}`, icon: BookOpenText, exact: true },
  { label: t('nav.editor'), to: `/projects/${activeProjectId.value}/editor`, icon: FilePenLine, exact: true },
  { label: t('nav.graph'), to: `/projects/${activeProjectId.value}/graph`, icon: GitFork, exact: true }
] : [])

const settingsItems = computed(() => [
  { label: t('nav.providers'), to: '/settings/providers', icon: Boxes },
  { label: t('nav.models'), to: '/settings/models', icon: Cpu },
  { label: t('nav.agents'), to: '/settings/agents', icon: Bot },
  { label: t('nav.indexMaintenance'), to: '/settings/index-maintenance', icon: DatabaseZap }
])

const desktopGroups = computed<AppNavigationGroup[]>(() => {
  const groups: AppNavigationGroup[] = [{ label: t('nav.creation'), items: creativeItems.value }]
  if (projectItems.value.length > 0) groups.push({ label: t('nav.currentProject'), items: projectItems.value })
  if (!route.path.startsWith('/settings')) groups.push({ label: t('nav.settings'), items: settingsItems.value })
  return groups
})

const mobileGroups = computed<AppNavigationGroup[]>(() => [
  { label: t('nav.creation'), items: [...creativeItems.value, ...projectItems.value] },
  { label: t('nav.settings'), items: settingsItems.value }
])

const mobilePrimaryItems = computed<AppNavigationGroup['items']>(() => [
  { label: t('nav.dashboard'), to: '/', icon: Gauge, exact: true },
  { label: t('nav.projects'), to: '/projects', icon: FolderOpen, exact: true },
  { label: t('nav.newProject'), to: '/projects/new', icon: Plus, exact: true }
])

const activeSectionLabel = computed(() => {
  const items = [...creativeItems.value, ...projectItems.value, ...settingsItems.value]
  return items.find((item) => isRouteActive(route.path, item.to, 'exact' in item && item.exact === true))?.label || t('product.tagline')
})

onMounted(() => {
  workspace.hydrateOpenedProjects()
  if (projectStore.items.length === 0 && !projectStore.listRequest.loading) {
    void projectStore.load()
      .then(() => workspace.syncOpenedProjects(projectStore.items))
      .catch((error) => console.error('[AeonEchoes Shell] Failed to load project navigation data.', error))
  }
})

watch(
  [activeProjectId, projects],
  ([projectId, projectList]) => {
    workspace.syncOpenedProjects(projectList)
    if (!projectId || openedProjects.value.some((project) => project.id === projectId)) return
    const project = projectList.find((item) => item.id === projectId)
    if (project) workspace.openProject(project)
  },
  { immediate: true }
)

watch(() => route.fullPath, () => {
  mobileMenuOpen.value = false
})

function openProject(projectId: string) {
  const project = projects.value.find((item) => item.id === projectId) || openedProjects.value.find((item) => item.id === projectId)
  if (!project) {
    console.error('[AeonEchoes Shell] Unable to navigate to an unknown project.', { projectId })
    return
  }
  workspace.openProject(project)
  navigateTo(`/projects/${projectId}`)
}

function closeProject(projectId: string) {
  workspace.closeProject(projectId)
  if (activeProjectId.value === projectId) navigateTo('/projects')
}
</script>

<template>
  <LayoutAppShell>
    <template #sidebar>
      <LayoutAppSidebar :label="t('nav.primaryNavigation')">
        <div class="flex h-topbar items-center gap-3 border-b border-border px-4">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center border border-foreground bg-foreground text-background">
            <BookOpenText class="h-4 w-4" aria-hidden="true" />
          </div>
          <div class="min-w-0">
            <p class="truncate text-sm font-bold tracking-tight">{{ t('product.name') }}</p>
            <p class="truncate text-[11px] text-muted-foreground">{{ t('product.tagline') }}</p>
          </div>
        </div>

        <div class="border-b border-border p-3">
          <ProjectSwitcher
            :projects="openedProjects"
            :active-project-id="activeProjectId"
            :label="t('nav.projectSwitcher')"
            :empty-label="t('nav.emptyOpenedProjects')"
            :close-label="t('nav.closeProject')"
            @select="openProject"
            @close="closeProject"
          />
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto px-3 py-5 subtle-scrollbar">
          <AppNavigation :groups="desktopGroups" :current-path="route.path" :label="t('nav.primaryNavigation')" />
        </div>

        <div class="border-t border-border p-3">
          <div class="flex items-center justify-between gap-2">
            <AppLanguageSwitcher class="min-w-0 flex-1" />
            <AppThemeToggle />
          </div>
        </div>
      </LayoutAppSidebar>
    </template>

    <template #topbar>
      <LayoutAppTopbar start-class="flex min-w-0 items-center gap-3">
        <template #start>
          <button
            type="button"
            class="focus-ring flex h-10 w-10 shrink-0 items-center justify-center border border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground lg:hidden"
            :aria-label="t('nav.openMenu')"
            @click="mobileMenuOpen = true"
          >
            <Menu class="h-5 w-5" aria-hidden="true" />
          </button>
          <div class="min-w-0">
            <p class="truncate text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">{{ activeProjectId ? t('nav.currentProject') : t('product.name') }}</p>
            <p class="truncate text-sm font-semibold text-foreground">{{ activeSectionLabel }}</p>
          </div>
        </template>

        {{ t('product.tagline') }}

        <template #end>
          <ProjectSwitcher
            v-if="activeProjectId"
            class="hidden w-56 md:block lg:hidden"
            :projects="openedProjects"
            :active-project-id="activeProjectId"
            :label="t('nav.projectSwitcher')"
            :empty-label="t('nav.emptyOpenedProjects')"
            :close-label="t('nav.closeProject')"
            @select="openProject"
            @close="closeProject"
          />
          <AppLanguageSwitcher compact class="lg:hidden" />
          <AppThemeToggle compact class="lg:hidden" />
        </template>
      </LayoutAppTopbar>
    </template>

    <LayoutAppMainContainer id="main-content" tabindex="-1">
      <slot />
    </LayoutAppMainContainer>

    <template #mobile>
      <nav class="fixed inset-x-0 bottom-0 z-40 grid h-16 grid-cols-4 border-t border-border bg-background px-2 pb-[env(safe-area-inset-bottom)] lg:hidden" :aria-label="t('nav.mobileNavigation')">
        <NuxtLink
          v-for="item in mobilePrimaryItems"
          :key="item.to"
          :to="item.to"
          :aria-current="isRouteActive(route.path, item.to, item.exact) ? 'page' : undefined"
          :class="[
            'focus-ring flex min-w-0 flex-col items-center justify-center gap-1 text-[11px] font-semibold transition-colors',
            isRouteActive(route.path, item.to, item.exact) ? 'text-foreground' : 'text-muted-foreground'
          ]"
        >
          <component :is="item.icon" class="h-4 w-4" aria-hidden="true" />
          <span class="truncate">{{ item.label }}</span>
        </NuxtLink>
        <button
          type="button"
          class="focus-ring flex min-w-0 flex-col items-center justify-center gap-1 text-[11px] font-semibold text-muted-foreground transition-colors hover:text-foreground"
          :aria-label="t('nav.openSettingsNavigation')"
          @click="mobileMenuOpen = true"
        >
          <Settings2 class="h-4 w-4" aria-hidden="true" />
          <span class="truncate">{{ t('nav.settings') }}</span>
        </button>
      </nav>

      <LayoutMobileNavSheet
        v-model:open="mobileMenuOpen"
        :title="t('nav.navigation')"
        :description="t('nav.mobileMenuDescription')"
        side="left"
        :aria-label="t('nav.openMenu')"
      >
        <div class="space-y-6">
          <ProjectSwitcher
            :projects="openedProjects"
            :active-project-id="activeProjectId"
            :label="t('nav.projectSwitcher')"
            :empty-label="t('nav.emptyOpenedProjects')"
            :close-label="t('nav.closeProject')"
            @select="openProject"
            @close="closeProject"
          />
          <AppNavigation :groups="mobileGroups" :current-path="route.path" :label="t('nav.mobileNavigation')" />
          <div class="grid grid-cols-[1fr_auto] gap-2 border-t border-border pt-4">
            <AppLanguageSwitcher />
            <AppThemeToggle compact />
          </div>
        </div>
      </LayoutMobileNavSheet>
    </template>
  </LayoutAppShell>
</template>
