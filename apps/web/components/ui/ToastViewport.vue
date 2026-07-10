<script setup lang="ts">
import { useToast } from '~/shared/composables/useToast'

const { messages, remove } = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed inset-x-3 bottom-20 z-[80] flex flex-col items-end gap-2 sm:left-auto sm:right-4 sm:top-4 sm:bottom-auto sm:w-[24rem]" :aria-label="$t('ui.toast.region')">
      <TransitionGroup
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="translate-y-2 opacity-0 sm:-translate-y-2"
        enter-to-class="translate-y-0 opacity-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <UiToast v-for="message in messages" :key="message.id" :message="message" @dismiss="remove" />
      </TransitionGroup>
    </div>
  </Teleport>
</template>
