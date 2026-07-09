<script setup lang="ts">
import { Plus, Trash2 } from '@lucide/vue'
import { cn } from '~/lib/utils'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: string[]
    label?: string
    addLabel?: string
    removeLabel?: string
    itemPlaceholder?: string
    disabled?: boolean
    minItems?: number
    class?: string
  }>(),
  {
    label: undefined,
    addLabel: undefined,
    removeLabel: undefined,
    itemPlaceholder: undefined,
    disabled: false,
    minItems: 0
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

function updateItem(index: number, value: string) {
  const next = [...props.modelValue]
  next[index] = value
  emit('update:modelValue', next)
}

function addItem() {
  if (props.disabled) return
  emit('update:modelValue', [...props.modelValue, ''])
}

function removeItem(index: number) {
  if (props.disabled || props.modelValue.length <= props.minItems) return
  emit('update:modelValue', props.modelValue.filter((_, itemIndex) => itemIndex !== index))
}
</script>

<template>
  <div :class="cn('min-w-0 space-y-3', props.class)">
    <div class="flex items-center justify-between gap-3">
      <p v-if="label" class="field-label text-foreground">{{ label }}</p>
      <UiButton type="button" variant="outline" size="sm" :disabled="disabled" @click="addItem">
        <Plus class="h-4 w-4" aria-hidden="true" />
        {{ addLabel || t('ui.editableList.add') }}
      </UiButton>
    </div>

    <div v-if="modelValue.length === 0" class="surface-muted rounded-xl px-3 py-3 text-sm text-muted-foreground">
      {{ t('ui.editableList.empty') }}
    </div>

    <ul v-else class="space-y-2">
      <li v-for="(item, index) in modelValue" :key="index" class="flex min-w-0 items-center gap-2">
        <UiInput
          :model-value="item"
          :disabled="disabled"
          :placeholder="itemPlaceholder || t('ui.editableList.itemPlaceholder')"
          :aria-label="label ? `${label} ${index + 1}` : t('ui.editableList.itemLabel', { index: index + 1 })"
          @update:model-value="updateItem(index, $event)"
        />
        <UiButton
          type="button"
          variant="ghost"
          size="icon"
          :disabled="disabled || modelValue.length <= minItems"
          :aria-label="removeLabel || t('ui.editableList.remove')"
          @click="removeItem(index)"
        >
          <Trash2 class="h-4 w-4" aria-hidden="true" />
        </UiButton>
      </li>
    </ul>
  </div>
</template>
