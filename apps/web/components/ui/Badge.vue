<script setup lang="ts">
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '~/lib/utils'

const badgeVariants = cva(
  'inline-flex max-w-full min-w-0 items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium leading-5 shadow-sm shadow-black/[0.02]',
  {
    variants: {
      tone: {
        info: 'border-state-info-border bg-state-info-surface text-state-info-foreground',
        success: 'border-state-success-border bg-state-success-surface text-state-success-foreground',
        warning: 'border-state-warning-border bg-state-warning-surface text-state-warning-foreground',
        danger: 'border-state-danger-border bg-state-danger-surface text-state-danger-foreground',
        neutral: 'border-border bg-surface text-foreground',
        muted: 'border-border bg-muted text-muted-foreground dark:bg-muted/70'
      }
    },
    defaultVariants: {
      tone: 'info'
    }
  }
)

type BadgeVariants = VariantProps<typeof badgeVariants>
type LegacyVariant = 'default' | 'violet' | 'gold' | 'rose' | 'muted' | 'success'

const props = withDefaults(
  defineProps<{
    tone?: BadgeVariants['tone']
    variant?: LegacyVariant
    class?: string
  }>(),
  {
    tone: undefined,
    variant: undefined
  }
)

const resolvedTone = computed<BadgeVariants['tone']>(() => {
  if (props.tone) return props.tone
  const variants: Record<LegacyVariant, BadgeVariants['tone']> = {
    default: 'info',
    violet: 'info',
    gold: 'warning',
    rose: 'danger',
    muted: 'muted',
    success: 'success'
  }
  return props.variant ? variants[props.variant] : 'info'
})
</script>

<template>
  <span :class="cn(badgeVariants({ tone: resolvedTone }), props.class)">
    <slot />
  </span>
</template>
