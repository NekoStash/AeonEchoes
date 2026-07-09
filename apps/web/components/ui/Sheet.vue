<script setup lang="ts">
import { X } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    ariaLabel?: string
    side?: 'right' | 'left'
    class?: string
  }>(),
  {
    description: '',
    ariaLabel: undefined,
    side: 'right'
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const titleId = `sheet-title-${Math.random().toString(36).slice(2, 10)}`
const descriptionId = `sheet-description-${Math.random().toString(36).slice(2, 10)}`
const sheetRef = ref<HTMLElement | null>(null)
let restoreFocusElement: HTMLElement | null = null
let lockedScroll = false

const resolvedAriaLabel = computed(() => props.ariaLabel || (!props.title ? t('ui.sheet.label') : undefined))

function close() {
  emit('update:open', false)
}

function updateScrollLock(locked: boolean) {
  if (!import.meta.client || lockedScroll === locked) return
  const body = document.body
  const currentCount = Number(body.dataset.aeonScrollLocks || '0')

  if (locked) {
    if (currentCount === 0) {
      body.dataset.aeonOriginalOverflow = body.style.overflow
      body.style.overflow = 'hidden'
    }
    body.dataset.aeonScrollLocks = String(currentCount + 1)
    lockedScroll = true
    return
  }

  const nextCount = Math.max(0, currentCount - 1)
  if (nextCount === 0) {
    body.style.overflow = body.dataset.aeonOriginalOverflow || ''
    delete body.dataset.aeonOriginalOverflow
    delete body.dataset.aeonScrollLocks
  } else {
    body.dataset.aeonScrollLocks = String(nextCount)
  }
  lockedScroll = false
}

function focusSheet() {
  const sheet = sheetRef.value
  if (!sheet) return
  const focusable = sheet.querySelector<HTMLElement>('[autofocus], button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')
  ;(focusable || sheet).focus()
}

function restoreFocus() {
  if (!import.meta.client || !restoreFocusElement) return
  restoreFocusElement.focus({ preventScroll: true })
  restoreFocusElement = null
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) close()
}

watch(
  () => props.open,
  async (isOpen) => {
    if (!import.meta.client) return
    if (isOpen) {
      restoreFocusElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
      document.addEventListener('keydown', handleKeydown)
      updateScrollLock(true)
      await nextTick()
      focusSheet()
      return
    }

    document.removeEventListener('keydown', handleKeydown)
    updateScrollLock(false)
    await nextTick()
    restoreFocus()
  },
  { immediate: true }
)

onUnmounted(() => {
  if (!import.meta.client) return
  document.removeEventListener('keydown', handleKeydown)
  updateScrollLock(false)
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm" @click="close" />
    </Transition>
    <Transition
      enter-active-class="transition duration-200 ease-out"
      :enter-from-class="side === 'right' ? 'translate-x-full opacity-0' : '-translate-x-full opacity-0'"
      enter-to-class="translate-x-0 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="translate-x-0 opacity-100"
      :leave-to-class="side === 'right' ? 'translate-x-full opacity-0' : '-translate-x-full opacity-0'"
    >
      <aside
        v-if="open"
        ref="sheetRef"
        :class="
          cn(
            'fixed top-0 z-50 flex h-[100dvh] w-[min(94vw,360px)] max-w-full flex-col border-border bg-card text-card-foreground shadow-2xl shadow-black/25',
            side === 'right' ? 'right-0 border-l' : 'left-0 border-r',
            props.class
          )
        "
        role="dialog"
        aria-modal="true"
        :aria-label="resolvedAriaLabel"
        :aria-labelledby="title ? titleId : undefined"
        :aria-describedby="description ? descriptionId : undefined"
        tabindex="-1"
        @click.stop
      >
        <div v-if="title || description" class="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 py-4">
          <div class="min-w-0 flex-1">
            <h2 v-if="title" :id="titleId" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" :id="descriptionId" class="mt-1 break-words text-sm text-muted-foreground">{{ description }}</p>
          </div>
          <button type="button" class="shrink-0 rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring" :aria-label="t('actions.close')" @click="close">
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div :class="cn('min-h-0 flex-1 overflow-y-auto subtle-scrollbar', title || description ? 'px-4 py-4 sm:px-5 sm:py-5' : 'px-4 py-4')">
          <div v-if="!title && !description" class="mb-3 flex justify-end">
            <button type="button" class="shrink-0 rounded-lg p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring" :aria-label="t('actions.close')" @click="close">
              <X class="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
          <slot />
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>
