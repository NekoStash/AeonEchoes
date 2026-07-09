<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    description?: string
    side?: 'left' | 'right'
    ariaLabel?: string
    class?: string
  }>(),
  {
    title: '',
    description: '',
    side: 'left',
    ariaLabel: undefined
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const openModel = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})
</script>

<template>
  <UiSheet
    v-model:open="openModel"
    :title="title"
    :description="description"
    :side="side"
    :aria-label="ariaLabel || title || undefined"
    :class="cn('w-[min(94vw,360px)] p-0 lg:hidden', props.class)"
  >
    <slot />
  </UiSheet>
</template>
