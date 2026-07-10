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
  side?: 'right' | 'left'
  closeOnBackdrop?: boolean
  class?: string
}>(), {
  description: '',
  ariaLabel: undefined,
  side: 'right',
  closeOnBackdrop: true
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const titleId = `sheet-title-${useId()}`
const descriptionId = `sheet-description-${useId()}`
const sheetRef = ref<HTMLElement | null>(null)
const openRef = toRef(props, 'open')
const resolvedAriaLabel = computed(() => props.ariaLabel || (!props.title ? t('ui.sheet.label') : undefined))

function close() {
  emit('update:open', false)
}

useModalFocus(openRef, sheetRef, close)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" data-aeon-overlay-layer="true">
    <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
      <div class="fixed inset-0 z-50 bg-black/72" @click="closeOnBackdrop && close()" />
    </Transition>
    <Transition enter-active-class="transition duration-150 ease-out" :enter-from-class="side === 'right' ? 'translate-x-full' : '-translate-x-full'" enter-to-class="translate-x-0" leave-active-class="transition duration-100 ease-in" leave-from-class="translate-x-0" :leave-to-class="side === 'right' ? 'translate-x-full' : '-translate-x-full'">
      <aside
        ref="sheetRef"
        :class="cn('fixed top-0 z-50 flex h-[100dvh] w-[min(92vw,380px)] max-w-full flex-col border-border bg-card text-card-foreground', side === 'right' ? 'right-0 border-l' : 'left-0 border-r', props.class)"
        role="dialog"
        aria-modal="true"
        :aria-label="resolvedAriaLabel"
        :aria-labelledby="title ? titleId : undefined"
        :aria-describedby="description ? descriptionId : undefined"
        tabindex="-1"
        @click.stop
      >
        <div class="flex shrink-0 items-start justify-between gap-4 border-b border-border px-4 py-4">
          <div class="min-w-0 flex-1">
            <h2 v-if="title" :id="titleId" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
            <p v-if="description" :id="descriptionId" class="mt-1 break-words text-sm leading-6 text-muted-foreground">{{ description }}</p>
            <span v-if="!title && !description" class="sr-only">{{ resolvedAriaLabel }}</span>
          </div>
          <button type="button" class="focus-ring flex h-9 w-9 shrink-0 items-center justify-center text-muted-foreground hover:bg-muted hover:text-foreground" :aria-label="t('actions.close')" @click="close">
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-4 subtle-scrollbar"><slot /></div>
        <div v-if="$slots.footer" class="shrink-0 border-t border-border p-4"><slot name="footer" /></div>
      </aside>
    </Transition>
    </div>
  </Teleport>
</template>
