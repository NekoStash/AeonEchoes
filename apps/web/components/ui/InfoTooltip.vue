<script setup lang="ts">
import { Info } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    text: string
    label?: string
    id?: string
    side?: 'top' | 'bottom'
    class?: string
  }>(),
  {
    label: '',
    side: 'top'
  }
)

const open = ref(false)
const generatedId = `info-tooltip-${Math.random().toString(36).slice(2, 10)}`
const tooltipId = computed(() => props.id || generatedId)
const ariaLabel = computed(() => props.label || props.text.split('\n')[0] || 'Info')

function show() {
  open.value = true
}

function hide() {
  open.value = false
}

function toggle() {
  open.value = !open.value
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    hide()
  }
}
</script>

<template>
  <span
    :class="cn('relative inline-flex shrink-0 align-middle', props.class)"
    @mouseenter="show"
    @mouseleave="hide"
    @focusin="show"
    @focusout="hide"
    @keydown="handleKeydown"
  >
    <button
      type="button"
      class="focus-ring inline-flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      :aria-label="ariaLabel"
      :aria-describedby="open ? tooltipId : undefined"
      :aria-expanded="open"
      @click.stop="toggle"
    >
      <Info class="h-3.5 w-3.5" aria-hidden="true" />
    </button>
    <span
      v-show="open"
      :id="tooltipId"
      role="tooltip"
      :class="cn(
        'absolute z-50 w-72 max-w-[min(18rem,calc(100vw-2rem))] rounded-xl border border-border bg-popover px-3 py-2 text-left text-xs leading-5 text-popover-foreground shadow-xl shadow-black/10 whitespace-pre-line',
        side === 'top' ? 'bottom-full left-1/2 mb-2 -translate-x-1/2' : 'left-1/2 top-full mt-2 -translate-x-1/2'
      )"
    >
      {{ text }}
    </span>
  </span>
</template>
