<script setup lang="ts">
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '~/lib/utils'

const badgeVariants = cva('inline-flex max-w-full min-w-0 items-center gap-1 rounded-sm border px-2 py-0.5 text-xs font-semibold leading-5', {
  variants: {
    tone: {
      info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
      success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
      warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
      danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
      neutral: 'border-border bg-surface text-foreground',
      muted: 'border-border bg-muted text-muted-foreground'
    }
  },
  defaultVariants: { tone: 'info' }
})

type BadgeVariants = VariantProps<typeof badgeVariants>

const props = withDefaults(defineProps<{
  tone?: BadgeVariants['tone']
  class?: string
}>(), {
  tone: 'info'
})
</script>

<template><span :class="cn(badgeVariants({ tone }), props.class)"><slot /></span></template>
