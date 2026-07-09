<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    class?: string
    contentClass?: string
  }>(),
  {}
)
</script>

<template>
  <div :class="cn('min-h-screen bg-background text-foreground', props.class)">
    <div class="flex min-h-screen">
      <slot name="sidebar" />
      <div :class="cn('min-w-0 flex-1', props.contentClass)">
        <slot name="topbar" />
        <slot />
      </div>
    </div>
    <slot name="mobile" />
  </div>
</template>
