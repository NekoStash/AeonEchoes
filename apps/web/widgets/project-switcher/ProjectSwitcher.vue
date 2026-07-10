<script setup lang="ts">
import { BookOpen, ChevronDown, X } from '@lucide/vue'
import { cn } from '~/lib/utils'

interface ProjectOption {
  id: string
  title: string
}

const props = withDefaults(
  defineProps<{
    projects: ProjectOption[]
    activeProjectId?: string
    label: string
    emptyLabel: string
    closeLabel: string
    compact?: boolean
    class?: string
  }>(),
  {
    activeProjectId: '',
    compact: false
  }
)

const emit = defineEmits<{
  select: [projectId: string]
  close: [projectId: string]
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const menuId = useId()

const activeProject = computed(() => props.projects.find((project) => project.id === props.activeProjectId))
const triggerLabel = computed(() => activeProject.value?.title || props.label)

function menuItems() {
  return Array.from(root.value?.querySelectorAll<HTMLElement>('[data-project-option]') || [])
}

async function openMenu() {
  open.value = true
  await nextTick()
  menuItems()[0]?.focus()
}

function closeMenu(restoreFocus = false) {
  open.value = false
  if (restoreFocus) nextTick(() => trigger.value?.focus())
}

function selectProject(projectId: string) {
  emit('select', projectId)
  closeMenu(true)
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    openMenu()
  }
}

function handleMenuKeydown(event: KeyboardEvent) {
  const items = menuItems()
  if (items.length === 0) return
  const currentIndex = Math.max(0, items.indexOf(document.activeElement as HTMLElement))
  let nextIndex: number | null = null

  if (event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % items.length
  if (event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + items.length) % items.length
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = items.length - 1
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMenu(true)
    return
  }
  if (nextIndex === null) return
  event.preventDefault()
  items[nextIndex]?.focus()
}

function handleDocumentClick(event: MouseEvent) {
  if (!root.value?.contains(event.target as Node)) closeMenu()
}

onMounted(() => document.addEventListener('click', handleDocumentClick))
onBeforeUnmount(() => document.removeEventListener('click', handleDocumentClick))
</script>

<template>
  <div ref="root" :class="cn('relative min-w-0', props.class)">
    <button
      ref="trigger"
      type="button"
      :aria-expanded="open"
      :aria-controls="open ? menuId : undefined"
      aria-haspopup="menu"
      :aria-label="compact ? triggerLabel : undefined"
      :title="compact ? triggerLabel : undefined"
      :class="cn(
        'focus-ring flex h-10 w-full min-w-0 items-center gap-2 border border-border bg-background px-3 text-left text-sm transition-colors hover:bg-muted',
        compact && 'w-10 justify-center px-0'
      )"
      @click.stop="open ? closeMenu() : openMenu()"
      @keydown="handleTriggerKeydown"
    >
      <BookOpen class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
      <span v-if="!compact" class="min-w-0 flex-1 truncate">{{ triggerLabel }}</span>
      <ChevronDown v-if="!compact" :class="cn('h-4 w-4 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180')" aria-hidden="true" />
    </button>

    <div
      v-if="open"
      :id="menuId"
      role="menu"
      class="absolute left-0 top-full z-50 mt-1 w-[min(20rem,calc(100vw-2rem))] border border-border bg-popover p-1 text-popover-foreground"
      @keydown="handleMenuKeydown"
    >
      <p v-if="projects.length === 0" class="px-3 py-3 text-sm leading-6 text-muted-foreground" role="status">
        {{ emptyLabel }}
      </p>
      <div v-for="project in projects" :key="project.id" class="flex min-w-0 items-center gap-1">
        <button
          type="button"
          role="menuitem"
          data-project-option
          :class="cn(
            'focus-ring min-w-0 flex-1 truncate px-3 py-2 text-left text-sm transition-colors hover:bg-muted',
            project.id === activeProjectId && 'bg-foreground text-background hover:bg-foreground/90'
          )"
          @click="selectProject(project.id)"
        >
          {{ project.title }}
        </button>
        <button
          type="button"
          class="focus-ring flex h-9 w-9 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          :aria-label="closeLabel.replace('{title}', project.title)"
          @click.stop="emit('close', project.id)"
        >
          <X class="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
  </div>
</template>
