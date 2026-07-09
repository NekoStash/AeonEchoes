<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    sticky?: boolean
    class?: string
    startClass?: string
    endClass?: string
  }>(),
  {
    sticky: true
  }
)
</script>

<template>
  <header
    :class="
      cn(
        'z-30 flex min-h-topbar items-center justify-between gap-3 border-b border-border bg-background/90 px-3 py-2 backdrop-blur sm:px-4 lg:h-topbar lg:px-6 lg:py-0',
        sticky && 'sticky top-0',
        props.class
      )
    "
  >
    <div :class="cn('min-w-0 flex-1', props.startClass)">
      <slot name="start" />
    </div>
    <div v-if="$slots.default" class="hidden min-w-0 flex-1 text-sm text-muted-foreground lg:block">
      <slot />
    </div>
    <div :class="cn('flex shrink-0 items-center gap-2', props.endClass)">
      <slot name="end" />
    </div>
  </header>
</template>
