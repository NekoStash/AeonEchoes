<script setup lang="ts">
import { Bot, BookOpen, FileText, FolderOpen, GitFork, Home, Menu, PlusCircle, Settings, X } from '@lucide/vue'
import { storeToRefs } from 'pinia'

const route = useRoute()
const { t } = useI18n()
const workspace = useWorkspaceStore()
const { openedProjects, projects } = storeToRefs(workspace)

const mobileMenuOpen = ref(false)

onMounted(() => {
  workspace.hydrateOpenedProjects()
  if (workspace.projects.length === 0) {
    workspace.loadDashboard()
  }
})

const navigation = computed(() => [
  { label: t('nav.dashboard'), to: '/', icon: Home, active: route.path === '/' },
  { label: t('nav.projects'), to: '/projects', icon: FolderOpen, active: route.path === '/projects' },
  { label: t('nav.models'), to: '/admin/models', icon: Settings, active: route.path.startsWith('/admin/models') },
  { label: t('nav.agents'), to: '/admin/agents', icon: Bot, active: route.path.startsWith('/admin/agents') },
  { label: t('nav.newProject'), to: '/projects/new', icon: PlusCircle, active: route.path === '/projects/new' }
])

const activeProjectId = computed(() => {
  const matched = route.path.match(/^\/projects\/([^/]+)/)
  if (!matched || matched[1] === 'new') return ''
  return matched[1]
})

watch(
  [activeProjectId, projects],
  ([projectId, projectList]) => {
    if (!projectId || openedProjects.value.some((project) => project.id === projectId)) {
      return
    }

    const project = projectList.find((item) => item.id === projectId)
    if (project) {
      workspace.openProject(project)
    }
  },
  { immediate: true }
)

watch(
  () => route.fullPath,
  () => {
    mobileMenuOpen.value = false
  }
)

function isProjectActive(projectId: string) {
  return route.path === `/projects/${projectId}`
}

function isProjectSectionActive(projectId: string) {
  return route.path.startsWith(`/projects/${projectId}`)
}

function isEditorActive(projectId: string) {
  return route.path === `/projects/${projectId}/editor`
}

function isGraphActive(projectId: string) {
  return route.path === `/projects/${projectId}/graph`
}

function closeOpenedProject(projectId: string) {
  workspace.closeProject(projectId)
}
</script>

<template>
  <LayoutAppShell>
    <template #sidebar>
      <LayoutAppSidebar :label="t('nav.openMenu')">
        <div class="flex h-topbar items-center gap-3 border-b border-border px-5">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <BookOpen class="h-4 w-4 shrink-0" aria-hidden="true" />
          </div>
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold">{{ t('product.name') }}</p>
            <p class="truncate text-xs text-muted-foreground">{{ t('product.tagline') }}</p>
          </div>
        </div>

        <div class="flex min-h-0 flex-1 flex-col px-3 py-4">
          <nav class="space-y-1" :aria-label="t('nav.openMenu')">
            <NuxtLink
              v-for="item in navigation"
              :key="item.to"
              :to="item.to"
              :aria-current="item.active ? 'page' : undefined"
              :class="[
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors focus-ring',
                item.active ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              ]"
            >
              <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
              <span class="truncate">{{ item.label }}</span>
            </NuxtLink>
          </nav>

          <div class="mt-5 border-t border-border pt-4">
            <div class="px-2 text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
              {{ t('nav.openedProjects') }}
            </div>

            <div v-if="openedProjects.length === 0" class="mt-3 rounded-lg border border-dashed border-border px-3 py-3 text-xs leading-5 text-muted-foreground">
              {{ t('nav.emptyOpenedProjects') }}
            </div>

            <div v-else class="mt-3 space-y-1.5">
              <div
                v-for="project in openedProjects"
                :key="project.id"
                :class="[
                  'rounded-lg border border-transparent p-1.5 transition-colors',
                  isProjectSectionActive(project.id) ? 'border-border bg-muted/70' : 'hover:bg-muted/45'
                ]"
              >
                <div class="flex items-center gap-1">
                  <NuxtLink
                    :to="`/projects/${project.id}`"
                    :aria-current="isProjectActive(project.id) ? 'page' : undefined"
                    :class="[
                      'min-w-0 flex-1 truncate rounded-md px-2 py-1.5 text-sm font-medium transition-colors focus-ring',
                      isProjectActive(project.id) ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'
                    ]"
                  >
                    {{ project.title }}
                  </NuxtLink>
                  <button
                    type="button"
                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-background hover:text-foreground focus-ring"
                    :aria-label="t('nav.closeProject', { title: project.title })"
                    @click.prevent.stop="closeOpenedProject(project.id)"
                  >
                    <X class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  </button>
                </div>
                <div class="ml-2 flex items-center gap-1 pb-1 pl-2 text-xs">
                  <NuxtLink
                    :to="`/projects/${project.id}/editor`"
                    :aria-current="isEditorActive(project.id) ? 'page' : undefined"
                    :class="[
                      'rounded-md px-2 py-1 transition-colors focus-ring',
                      isEditorActive(project.id) ? 'bg-background text-foreground' : 'text-muted-foreground hover:bg-background hover:text-foreground'
                    ]"
                  >
                    <FileText class="mr-1 inline h-3 w-3 shrink-0 align-[-2px]" aria-hidden="true" />
                    {{ t('nav.editor') }}
                  </NuxtLink>
                  <NuxtLink
                    :to="`/projects/${project.id}/graph`"
                    :aria-current="isGraphActive(project.id) ? 'page' : undefined"
                    :class="[
                      'rounded-md px-2 py-1 transition-colors focus-ring',
                      isGraphActive(project.id) ? 'bg-background text-foreground' : 'text-muted-foreground hover:bg-background hover:text-foreground'
                    ]"
                  >
                    <GitFork class="mr-1 inline h-3 w-3 shrink-0 align-[-2px]" aria-hidden="true" />
                    {{ t('nav.graph') }}
                  </NuxtLink>
                </div>
              </div>
            </div>
          </div>
        </div>
      </LayoutAppSidebar>
    </template>

    <template #topbar>
      <LayoutAppTopbar start-class="flex min-w-0 items-center gap-2 lg:hidden">
        <template #start>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-muted/70 hover:text-foreground focus-ring"
            :aria-label="t('nav.openMenu')"
            @click="mobileMenuOpen = true"
          >
            <Menu class="h-5 w-5 shrink-0" aria-hidden="true" />
          </button>
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <BookOpen class="h-4 w-4 shrink-0" aria-hidden="true" />
          </div>
          <div class="min-w-0">
            <span class="block truncate text-sm font-semibold leading-5">{{ t('product.name') }}</span>
            <span class="block truncate text-[11px] leading-4 text-muted-foreground sm:hidden">{{ t('product.tagline') }}</span>
          </div>
        </template>

        {{ t('product.tagline') }}

        <template #end>
          <AppLanguageSwitcher class="hidden sm:block" />
          <AppLanguageSwitcher compact class="sm:hidden" />
          <AppThemeToggle class="hidden sm:inline-flex" />
          <AppThemeToggle compact class="sm:hidden" />
        </template>
      </LayoutAppTopbar>
    </template>

    <LayoutAppMainContainer>
      <slot />
    </LayoutAppMainContainer>

    <template #mobile>
      <LayoutMobileNavSheet v-model:open="mobileMenuOpen" title="" description="" side="left" :aria-label="t('nav.openMenu')">
        <div class="space-y-5 px-1 pb-4">
          <div class="rounded-2xl border border-border bg-muted/35 p-3">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                <BookOpen class="h-4 w-4 shrink-0" aria-hidden="true" />
              </div>
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold">{{ t('product.name') }}</p>
                <p class="truncate text-xs text-muted-foreground">{{ t('product.tagline') }}</p>
              </div>
            </div>
          </div>
          <nav class="space-y-1" :aria-label="t('nav.openMenu')">
            <NuxtLink
              v-for="item in navigation"
              :key="item.to"
              :to="item.to"
              :aria-current="item.active ? 'page' : undefined"
              :class="[
                'flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-colors focus-ring',
                item.active ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              ]"
            >
              <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
              <span class="truncate">{{ item.label }}</span>
            </NuxtLink>
          </nav>

          <div class="border-t border-border pt-4">
            <div class="px-2 text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
              {{ t('nav.openedProjects') }}
            </div>

            <div v-if="openedProjects.length === 0" class="mt-3 rounded-lg border border-dashed border-border px-3 py-3 text-xs leading-5 text-muted-foreground">
              {{ t('nav.emptyOpenedProjects') }}
            </div>

            <div v-else class="mt-3 space-y-1.5">
              <div
                v-for="project in openedProjects"
                :key="project.id"
                :class="[
                  'rounded-lg border border-transparent p-1.5 transition-colors',
                  isProjectSectionActive(project.id) ? 'border-border bg-muted/70' : 'hover:bg-muted/45'
                ]"
              >
                <div class="flex items-center gap-1">
                  <NuxtLink
                    :to="`/projects/${project.id}`"
                    :aria-current="isProjectActive(project.id) ? 'page' : undefined"
                    :class="[
                      'min-w-0 flex-1 truncate rounded-md px-2 py-1.5 text-sm font-medium transition-colors focus-ring',
                      isProjectActive(project.id) ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'
                    ]"
                  >
                    {{ project.title }}
                  </NuxtLink>
                  <button
                    type="button"
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-background hover:text-foreground focus-ring"
                    :aria-label="t('nav.closeProject', { title: project.title })"
                    @click.prevent.stop="closeOpenedProject(project.id)"
                  >
                    <X class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  </button>
                </div>
                <div class="ml-2 flex flex-wrap items-center gap-1 pb-1 pl-2 text-xs">
                  <NuxtLink
                    :to="`/projects/${project.id}/editor`"
                    :aria-current="isEditorActive(project.id) ? 'page' : undefined"
                    :class="[
                      'rounded-md px-2 py-1.5 transition-colors focus-ring',
                      isEditorActive(project.id) ? 'bg-background text-foreground' : 'text-muted-foreground hover:bg-background hover:text-foreground'
                    ]"
                  >
                    <FileText class="mr-1 inline h-3 w-3 shrink-0 align-[-2px]" aria-hidden="true" />
                    {{ t('nav.editor') }}
                  </NuxtLink>
                  <NuxtLink
                    :to="`/projects/${project.id}/graph`"
                    :aria-current="isGraphActive(project.id) ? 'page' : undefined"
                    :class="[
                      'rounded-md px-2 py-1.5 transition-colors focus-ring',
                      isGraphActive(project.id) ? 'bg-background text-foreground' : 'text-muted-foreground hover:bg-background hover:text-foreground'
                    ]"
                  >
                    <GitFork class="mr-1 inline h-3 w-3 shrink-0 align-[-2px]" aria-hidden="true" />
                    {{ t('nav.graph') }}
                  </NuxtLink>
                </div>
              </div>
            </div>
          </div>
        </div>
      </LayoutMobileNavSheet>
    </template>
  </LayoutAppShell>
</template>
