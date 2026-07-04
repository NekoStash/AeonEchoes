<script setup lang="ts">
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '~/lib/utils'

const badgeVariants = cva('inline-flex max-w-full min-w-0 items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium leading-5 shadow-sm shadow-black/[0.02]', {
  variants: {
    variant: {
      default: 'border-sky-200 bg-sky-50 text-sky-800 dark:border-cyan-300/25 dark:bg-cyan-300/10 dark:text-cyan-100',
      violet: 'border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-300/25 dark:bg-violet-300/10 dark:text-violet-100',
      gold: 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-300/25 dark:bg-amber-300/10 dark:text-amber-100',
      rose: 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-300/25 dark:bg-rose-300/10 dark:text-rose-100',
      muted: 'border-border bg-muted text-muted-foreground dark:bg-muted/70',
      success: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-300/25 dark:bg-emerald-300/10 dark:text-emerald-100'
    }
  },
  defaultVariants: {
    variant: 'default'
  }
})

type BadgeVariants = VariantProps<typeof badgeVariants>

const props = withDefaults(
  defineProps<{
    variant?: BadgeVariants['variant']
    class?: string
  }>(),
  {
    variant: 'default'
  }
)
</script>

<template>
  <span :class="cn(badgeVariants({ variant }), props.class)">
    <slot />
  </span>
</template>
