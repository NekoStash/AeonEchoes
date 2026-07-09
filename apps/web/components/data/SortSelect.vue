<script setup lang="ts">
interface SortOption {
  label: string
  value: string
  description?: string
  disabled?: boolean
  disabledReason?: string
}

const { t } = useI18n()

withDefaults(
  defineProps<{
    modelValue?: string
    options: SortOption[]
    placeholder?: string
    class?: string
  }>(),
  {
    modelValue: '',
    placeholder: undefined
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <UiSelect
    :model-value="modelValue"
    :options="options"
    :placeholder="placeholder || t('ui.sort.placeholder')"
    :empty-text="t('ui.select.noResults')"
    :class="class"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
