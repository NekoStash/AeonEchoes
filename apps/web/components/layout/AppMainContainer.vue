<script setup lang="ts">
import { cn } from '~/lib/utils'

type ContainerSize = 'page' | 'panel' | 'readable' | 'full'
type ContainerPadding = 'none' | 'sm' | 'md' | 'lg'

const props = withDefaults(defineProps<{
  as?: string
  size?: ContainerSize
  padding?: ContainerPadding
  class?: string
}>(), {
  as: 'main',
  size: 'page',
  padding: 'lg'
})

const sizeClass = computed(() => ({
  page: 'max-w-layout-page',
  panel: 'max-w-layout-panel',
  readable: 'max-w-layout-readable',
  full: 'max-w-none'
})[props.size])

const paddingClass = computed(() => ({
  none: '',
  sm: 'px-3 py-4 sm:px-4',
  md: 'px-4 py-5 sm:px-6',
  lg: 'px-4 py-6 lg:px-8 2xl:px-10'
})[props.padding])
</script>

<template>
  <component :is="as" :class="cn('mx-auto w-full min-w-0', sizeClass, paddingClass, props.class)">
    <slot />
  </component>
</template>
