<script setup lang="ts">
import { Rows2, Rows3, Rows4 } from '@lucide/vue'
import { cn } from '~/lib/utils'

export type DataDensity = 'compact' | 'comfortable' | 'relaxed'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: DataDensity
    densities?: DataDensity[]
    label?: string
    class?: string
  }>(),
  {
    densities: () => ['compact', 'comfortable', 'relaxed']
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: DataDensity]
}>()

const iconByDensity = {
  compact: Rows2,
  comfortable: Rows3,
  relaxed: Rows4
}

function densityLabel(density: DataDensity) {
  return t(`ui.density.${density}`)
}
</script>

<template>
  <div :class="cn('inline-flex items-center gap-1 rounded-xl border border-border bg-surface-muted p-1', props.class)" role="group" :aria-label="label || t('ui.density.label')">
    <button
      v-for="density in densities"
      :key="density"
      type="button"
      :aria-pressed="modelValue === density"
      :title="densityLabel(density)"
      :class="cn('focus-ring inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-2 text-sm text-muted-foreground transition-colors hover:bg-surface hover:text-foreground', modelValue === density && 'bg-surface text-foreground shadow-sm')"
      @click="emit('update:modelValue', density)"
    >
      <component :is="iconByDensity[density]" class="h-4 w-4" aria-hidden="true" />
      <span class="sr-only">{{ densityLabel(density) }}</span>
    </button>
  </div>
</template>
