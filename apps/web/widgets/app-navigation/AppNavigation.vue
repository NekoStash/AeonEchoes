<script setup lang="ts">
import { cn } from '~/lib/utils'
import type { AppNavigationGroup } from './navigation'
import { isRouteActive } from './navigation'

const props = withDefaults(
  defineProps<{
    groups: AppNavigationGroup[]
    currentPath: string
    label: string
    compact?: boolean
    class?: string
  }>(),
  {
    compact: false
  }
)
</script>

<template>
  <nav :aria-label="label" :class="cn('space-y-6', props.class)">
    <section v-for="group in groups" :key="group.label" class="space-y-2">
      <h2 v-if="!compact" class="px-3 text-[0.68rem] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {{ group.label }}
      </h2>
      <div class="space-y-1">
        <NuxtLink
          v-for="item in group.items"
          :key="item.to"
          :to="item.to"
          :title="compact ? item.label : undefined"
          :aria-label="compact ? item.label : undefined"
          :aria-current="isRouteActive(currentPath, item.to, item.exact) ? 'page' : undefined"
          :class="cn(
            'focus-ring group relative flex min-h-10 items-center gap-3 border border-transparent px-3 py-2 text-sm font-medium transition-colors',
            compact && 'justify-center px-2',
            isRouteActive(currentPath, item.to, item.exact)
              ? 'border-border bg-foreground text-background'
              : 'text-muted-foreground hover:border-border hover:bg-muted hover:text-foreground'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
          <span v-if="!compact" class="min-w-0 flex-1 truncate">{{ item.label }}</span>
        </NuxtLink>
      </div>
    </section>
  </nav>
</template>
