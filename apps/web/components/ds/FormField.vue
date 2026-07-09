<script setup lang="ts">
import { cn } from '~/lib/utils'

const props = withDefaults(
  defineProps<{
    id?: string
    label?: string
    description?: string
    error?: string
    required?: boolean
    class?: string
  }>(),
  {
    id: undefined,
    label: undefined,
    description: undefined,
    error: undefined,
    required: false
  }
)

const generatedId = `field-${Math.random().toString(36).slice(2, 10)}`
const controlId = computed(() => props.id || generatedId)
const descriptionId = computed(() => `${controlId.value}-description`)
const errorId = computed(() => `${controlId.value}-error`)
</script>

<template>
  <div :class="cn('min-w-0 space-y-2', props.class)">
    <label v-if="label" :for="controlId" class="field-label text-foreground">
      {{ label }}
      <span v-if="required" class="text-state-danger" aria-hidden="true">*</span>
    </label>
    <p v-if="description" :id="descriptionId" class="text-xs leading-5 text-muted-foreground">{{ description }}</p>
    <slot :id="controlId" :describedby="[description ? descriptionId : '', error ? errorId : ''].filter(Boolean).join(' ') || undefined" :invalid="Boolean(error)" />
    <p v-if="error" :id="errorId" class="text-xs leading-5 text-state-danger" role="alert">{{ error }}</p>
  </div>
</template>
