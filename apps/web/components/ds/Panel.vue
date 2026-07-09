<script setup lang="ts">
import { cn } from '~/lib/utils'

type PanelPadding = 'none' | 'sm' | 'md' | 'lg'
type PanelTone = 'default' | 'muted' | 'elevated' | 'sunken'

const props = withDefaults(
  defineProps<{
    as?: string
    tone?: PanelTone
    padding?: PanelPadding
    interactive?: boolean
    class?: string
    bodyClass?: string
  }>(),
  {
    as: 'section',
    tone: 'default',
    padding: 'md',
    interactive: false
  }
)

const toneClass = computed(() => {
  const tones: Record<PanelTone, string> = {
    default: 'surface-panel',
    muted: 'surface-muted',
    elevated: 'surface-elevated',
    sunken: 'surface-sunken'
  }
  return tones[props.tone]
})

const paddingClass = computed(() => {
  const paddings: Record<PanelPadding, string> = {
    none: '',
    sm: 'p-3',
    md: 'p-4 sm:p-5',
    lg: 'p-5 sm:p-6'
  }
  return paddings[props.padding]
})
</script>

<template>
  <component
    :is="as"
    :class="cn('min-w-0 rounded-2xl', toneClass, interactive && 'transition-all hover:-translate-y-0.5 hover:border-primary/35 hover:shadow-md', props.class)"
  >
    <slot name="header" />
    <div :class="cn(paddingClass, props.bodyClass)">
      <slot />
    </div>
    <div v-if="$slots.footer" class="border-t border-border px-4 py-3 sm:px-5">
      <slot name="footer" />
    </div>
  </component>
</template>
