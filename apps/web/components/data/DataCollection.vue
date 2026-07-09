<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    loading?: boolean
    error?: string
    empty?: boolean
    noResults?: boolean
    title?: string
    description?: string
    loadingTitle?: string
    loadingDescription?: string
    emptyTitle?: string
    emptyDescription?: string
    noResultsTitle?: string
    noResultsDescription?: string
    class?: string
    bodyClass?: string
  }>(),
  {
    loading: false,
    error: '',
    empty: false,
    noResults: false
  }
)
</script>

<template>
  <section :class="cn('min-w-0 space-y-4', props.class)">
    <div v-if="title || description || $slots.toolbar" class="flex min-w-0 flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div v-if="title || description" class="min-w-0">
        <h2 v-if="title" class="break-words text-lg font-semibold text-foreground">{{ title }}</h2>
        <p v-if="description" class="mt-1 break-words text-sm leading-6 text-muted-foreground">{{ description }}</p>
      </div>
      <div v-if="$slots.toolbar" class="min-w-0 lg:shrink-0">
        <slot name="toolbar" />
      </div>
    </div>

    <slot v-if="$slots.filters" name="filters" />

    <slot v-if="loading" name="loading">
      <DataLoadingState :title="loadingTitle" :description="loadingDescription" />
    </slot>
    <slot v-else-if="error" name="error">
      <DataErrorState :description="error" />
    </slot>
    <slot v-else-if="empty" name="empty">
      <DataEmptyState :title="emptyTitle" :description="emptyDescription" />
    </slot>
    <slot v-else-if="noResults" name="no-results">
      <DataNoResultsState :title="noResultsTitle" :description="noResultsDescription" />
    </slot>
    <div v-else :class="cn('min-w-0', props.bodyClass)">
      <slot />
    </div>
  </section>
</template>
