<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = defineProps<{
  label: string
  value: string | number
  hint?: string
  tone?: 'cyan' | 'violet' | 'gold' | 'rose' | 'emerald'
}>()

const toneClass = computed(() => {
  const tones = {
    cyan: 'border-sky-200 from-sky-50 to-white text-sky-800 dark:border-cyan-300/20 dark:from-cyan-300/18 dark:to-cyan-300/5 dark:text-cyan-100',
    violet: 'border-violet-200 from-violet-50 to-white text-violet-800 dark:border-violet-300/20 dark:from-violet-300/18 dark:to-violet-300/5 dark:text-violet-100',
    gold: 'border-amber-200 from-amber-50 to-white text-amber-900 dark:border-amber-300/20 dark:from-amber-300/18 dark:to-amber-300/5 dark:text-amber-100',
    rose: 'border-rose-200 from-rose-50 to-white text-rose-800 dark:border-rose-300/20 dark:from-rose-300/18 dark:to-rose-300/5 dark:text-rose-100',
    emerald: 'border-emerald-200 from-emerald-50 to-white text-emerald-800 dark:border-emerald-300/20 dark:from-emerald-300/18 dark:to-emerald-300/5 dark:text-emerald-100'
  }
  return tones[props.tone || 'cyan']
})
</script>

<template>
  <div :class="cn('rounded-2xl border bg-gradient-to-br p-4 shadow-sm', toneClass)">
    <p class="text-xs uppercase tracking-[0.22em] text-current/65">{{ label }}</p>
    <p class="mt-3 text-2xl font-semibold text-current">{{ value }}</p>
    <p v-if="hint" class="mt-1 text-xs text-current/65">{{ hint }}</p>
  </div>
</template>
