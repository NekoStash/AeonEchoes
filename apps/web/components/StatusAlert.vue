<script setup lang="ts">
import type { ApiErrorState } from '~/lib/types'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    errors?: ApiErrorState[]
  }>(),
  {
    errors: () => []
  }
)

function errorKey(error: ApiErrorState) {
  return `${error.endpoint}:${error.message}`
}
</script>

<template>
  <div class="space-y-2">
    <UiAlert
      v-for="error in props.errors"
      :key="errorKey(error)"
      tone="warning"
      :title="`${t('apiError.title')} · ${error.endpoint}`"
      :description="error.message"
    />
  </div>
</template>

