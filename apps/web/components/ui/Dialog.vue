<script setup lang="ts">
import { X } from '@lucide/vue'
import { computed, ref, toRef, useId } from 'vue'
import { cn } from '~/lib/utils'
import { useModalFocus } from '~/shared/composables/useModalFocus'

const { t } = useI18n()
const props = withDefaults(defineProps<{
  open: boolean
  title: string
  description?: string
  ariaLabel?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  closeOnBackdrop?: boolean
  class?: string
}>(), {
  description: '',
  ariaLabel: undefined,
  size: 'md',
  closeOnBackdrop: true
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const titleId = `dialog-title-${useId()}`
const descriptionId = `dialog-description-${useId()}`
const dialogRef = ref<HTMLElement | null>(null)
const openRef = toRef(props, 'open')

const sizeClass = computed(() => ({
  sm: 'sm:max-w-md',
  md: 'sm:max-w-2xl',
  lg: 'sm:max-w-3xl',
  xl: 'sm:max-w-5xl',
  full: 'sm:max-w-[min(1200px,calc(100vw-2rem))]'
})[props.size])
const resolvedAriaLabel = computed(() => props.ariaLabel || (!props.title ? t('ui.dialog.label') : undefined))

function close() {
  emit('update:open', false)
}

useModalFocus(openRef, dialogRef, close)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" data-aeon-overlay-layer="true">
    <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
      <div class="fixed inset-0 z-50 bg-black/72" @click="closeOnBackdrop && close()" />
    </Transition>
    <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="translate-y-3 opacity-0" enter-to-class="translate-y-0 opacity-100" leave-active-class="transition duration-100 ease-in" leave-from-class="translate-y-0 opacity-100" leave-to-class="translate-y-3 opacity-0">
      <section
        ref="dialogRef"
        :class="cn('fixed inset-x-2 bottom-2 top-6 z-50 mx-auto flex w-auto flex-col border border-border bg-card sm:inset-x-4 sm:bottom-auto sm:top-1/2 sm:max-h-[min(86vh,900px)] sm:w-[calc(100vw-2rem)] sm:-translate-y-1/2', sizeClass, props.class)"
        role="dialog"
        aria-modal="true"
        :aria-label="resolvedAriaLabel"
        :aria-labelledby="title ? titleId : undefined"
        :aria-describedby="description ? descriptionId : undefined"
        tabindex="-1"
        @click.stop
      >
        <div class="flex shrink-0 items-start justify-between gap-4 border-b border-border px-4 py-4 sm:px-6">
          <div class="min-w-0">
            <h2 v-if="title" :id="titleId" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" :id="descriptionId" class="mt-1 break-words text-sm leading-6 text-muted-foreground">{{ description }}</p>
          </div>
          <button type="button" class="focus-ring flex h-9 w-9 shrink-0 items-center justify-center text-muted-foreground hover:bg-muted hover:text-foreground" :aria-label="t('actions.close')" @click="close">
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 subtle-scrollbar sm:px-6"><slot /></div>
        <div v-if="$slots.footer" class="shrink-0 border-t border-border px-4 py-3 sm:px-6"><slot name="footer" /></div>
      </section>
    </Transition>
    </div>
  </Teleport>
</template>
