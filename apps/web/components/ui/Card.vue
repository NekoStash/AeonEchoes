<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = defineProps<{
  class?: string
  interactive?: boolean
}>()
</script>

<template>
  <section
    :class="
      cn(
        'min-w-0 rounded-2xl border border-border bg-card text-card-foreground shadow-sm shadow-black/[0.03]',
        interactive && 'transition-all hover:-translate-y-0.5 hover:border-primary/35 hover:shadow-md',
        props.class
      )
    "
  >
    <slot />
  </section>
</template>
