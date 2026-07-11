<script setup lang="ts">
const { t } = useI18n()

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  tone?: 'default' | 'danger'
  loading?: boolean
  restoreFocus?: boolean
}>(), {
  description: '',
  confirmLabel: undefined,
  cancelLabel: undefined,
  tone: 'danger',
  loading: false,
  restoreFocus: true
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
  afterClose: []
}>()

const openModel = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})
</script>

<template>
  <UiDialog
    v-model:open="openModel"
    :title="title"
    :description="description"
    :restore-focus="restoreFocus"
    size="sm"
    @after-close="emit('afterClose')"
  >
    <slot />
    <template #footer>
      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <UiButton variant="outline" :disabled="loading" @click="openModel = false">
          {{ cancelLabel || t('actions.cancel') }}
        </UiButton>
        <UiButton :variant="tone === 'danger' ? 'destructive' : 'default'" :loading="loading" @click="emit('confirm')">
          {{ confirmLabel || t('actions.confirm') }}
        </UiButton>
      </div>
    </template>
  </UiDialog>
</template>
