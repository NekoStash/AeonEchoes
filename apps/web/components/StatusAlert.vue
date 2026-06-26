<script setup lang="ts">
import { AlertCircle } from '@lucide/vue'
import type { ApiErrorState } from '~/lib/types'

const { t } = useI18n()

defineProps<{
  errors?: ApiErrorState[]
}>()
</script>

<template>
  <div class="space-y-2">
    <div
      v-for="error in errors"
      :key="`${error.endpoint}:${error.message}`"
      class="flex items-start gap-3 rounded-xl border border-amber-300/40 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100"
    >
      <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
      <div>
        <p class="font-medium">{{ t('apiError.title') }} · {{ error.endpoint }}</p>
        <p class="mt-1 opacity-80">{{ error.message }}</p>
      </div>
    </div>
  </div>
</template>
