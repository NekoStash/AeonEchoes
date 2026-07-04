<script setup lang="ts">
import { X } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    description?: string
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
    class?: string
  }>(),
  {
    description: '',
    size: 'md'
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const titleId = `dialog-title-${Math.random().toString(36).slice(2, 10)}`
const descriptionId = `dialog-description-${Math.random().toString(36).slice(2, 10)}`

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'sm:max-w-md'
    case 'lg':
      return 'sm:max-w-3xl'
    case 'xl':
      return 'sm:max-w-5xl'
    case 'full':
      return 'sm:max-w-[min(1200px,calc(100vw-2rem))]'
    default:
      return 'sm:max-w-2xl'
  }
})

function close() {
  emit('update:open', false)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) close()
}

watch(
  () => props.open,
  (isOpen) => {
    if (!import.meta.client) return
    if (isOpen) document.addEventListener('keydown', handleKeydown)
    else document.removeEventListener('keydown', handleKeydown)
  },
  { immediate: true }
)

onUnmounted(() => {
  if (import.meta.client) document.removeEventListener('keydown', handleKeydown)
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
      <div v-if="open" class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm" @click="close" />
    </Transition>
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="translate-y-4 scale-95 opacity-0 sm:translate-y-3"
      enter-to-class="translate-y-0 scale-100 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="translate-y-0 scale-100 opacity-100"
      leave-to-class="translate-y-4 scale-95 opacity-0 sm:translate-y-3"
    >
      <section
        v-if="open"
        :class="cn('fixed inset-x-2 bottom-2 top-6 z-50 mx-auto flex w-auto flex-col rounded-2xl border border-border bg-card shadow-2xl sm:inset-x-4 sm:bottom-auto sm:top-1/2 sm:max-h-[min(86vh,900px)] sm:w-[calc(100vw-2rem)] sm:-translate-y-1/2', sizeClass, props.class)"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="description ? descriptionId : undefined"
        @click.stop
      >
        <div class="flex shrink-0 items-start justify-between gap-4 border-b border-border px-4 py-4 sm:px-6">
          <div class="min-w-0">
            <h2 :id="titleId" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" :id="descriptionId" class="mt-1 break-words text-sm leading-6 text-muted-foreground">{{ description }}</p>
          </div>
          <button type="button" class="focus-ring rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground" :aria-label="t('actions.close')" @click="close">
            <X class="h-4 w-4" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 subtle-scrollbar sm:px-6">
          <slot />
        </div>
        <div v-if="$slots.footer" class="shrink-0 border-t border-border px-4 py-3 sm:px-6">
          <slot name="footer" />
        </div>
      </section>
    </Transition>
  </Teleport>
</template>
