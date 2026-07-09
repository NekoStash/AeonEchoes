<script setup lang="ts">
import { Grid2X2, List, Table2 } from '@lucide/vue'
import { cn } from '~/lib/utils'

export type ViewMode = 'table' | 'grid' | 'list'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: ViewMode
    modes?: ViewMode[]
    label?: string
    class?: string
  }>(),
  {
    modes: () => ['table', 'grid']
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: ViewMode]
}>()

const iconByMode = {
  table: Table2,
  grid: Grid2X2,
  list: List
}

function modeLabel(mode: ViewMode) {
  return t(`ui.viewMode.${mode}`)
}
</script>

<template>
  <div :class="cn('inline-flex items-center gap-1 rounded-xl border border-border bg-surface-muted p-1', props.class)" role="group" :aria-label="label || t('ui.viewMode.label')">
    <button
      v-for="mode in modes"
      :key="mode"
      type="button"
      :aria-pressed="modelValue === mode"
      :title="modeLabel(mode)"
      :class="cn('focus-ring inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-2 text-sm text-muted-foreground transition-colors hover:bg-surface hover:text-foreground', modelValue === mode && 'bg-surface text-foreground shadow-sm')"
      @click="emit('update:modelValue', mode)"
    >
      <component :is="iconByMode[mode]" class="h-4 w-4" aria-hidden="true" />
      <span class="sr-only">{{ modeLabel(mode) }}</span>
    </button>
  </div>
</template>
