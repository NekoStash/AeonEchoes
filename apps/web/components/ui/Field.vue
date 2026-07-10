<script setup lang="ts">
import { computed, useId } from 'vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

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

const generatedId = useId()
const controlId = computed(() => props.id || `field-${generatedId}`)
const descriptionId = computed(() => `${controlId.value}-description`)
const errorId = computed(() => `${controlId.value}-error`)
const describedBy = computed(() => [props.description ? descriptionId.value : '', props.error ? errorId.value : ''].filter(Boolean).join(' ') || undefined)
</script>

<template>
  <div :class="cn('min-w-0 space-y-2', props.class)">
    <label v-if="label" :for="controlId" class="field-label">
      {{ label }}
      <span v-if="required" class="text-state-danger" aria-hidden="true">*</span>
      <span v-if="required" class="sr-only">{{ t('ui.field.required') }}</span>
    </label>
    <p v-if="description" :id="descriptionId" class="text-xs leading-5 text-muted-foreground">{{ description }}</p>
    <slot :id="controlId" :describedby="describedBy" :invalid="Boolean(error)" />
    <p v-if="error" :id="errorId" class="text-xs font-medium leading-5 text-state-danger" role="alert">{{ error }}</p>
  </div>
</template>
