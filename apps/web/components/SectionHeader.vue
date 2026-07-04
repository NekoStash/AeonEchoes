<script setup lang="ts">
defineProps<{
  eyebrow?: string
  title: string
  description?: string
}>()
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
    <div class="min-w-0 flex-1">
      <p v-if="eyebrow" class="truncate text-xs font-medium uppercase tracking-wide text-muted-foreground">{{ eyebrow }}</p>
      <h1 class="mt-1 max-w-5xl break-words text-2xl font-semibold tracking-tight text-foreground md:text-3xl">{{ title }}</h1>
      <p v-if="description" class="mt-2 max-w-4xl break-words text-sm leading-7 text-muted-foreground">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="flex w-full min-w-0 flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:justify-end lg:shrink-0">
      <slot name="actions" />
    </div>
  </div>
</template>
