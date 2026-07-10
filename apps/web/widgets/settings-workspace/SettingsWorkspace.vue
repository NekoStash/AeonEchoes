<script setup lang="ts">
import { Settings2 } from '@lucide/vue'
import { cn } from '~/lib/utils'

const props = withDefaults(defineProps<{
  title: string
  description: string
  eyebrow?: string
  layout?: 'document' | 'viewport'
}>(), {
  eyebrow: undefined,
  layout: 'document'
})

const { t } = useI18n()
</script>

<template>
  <PageShell
    :class="cn(
      'pb-8',
      layout === 'viewport' && 'lg:flex lg:h-[calc(100dvh-var(--layout-height-topbar)-3rem)] lg:min-h-0 lg:flex-col lg:space-y-0 lg:overflow-hidden lg:pb-0'
    )"
  >
    <header :class="cn('border-y border-border bg-foreground px-5 py-7 text-background sm:px-7 lg:px-9', layout === 'viewport' && 'lg:shrink-0 lg:py-5')">
      <div class="flex items-start gap-4">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center border border-background/30"><Settings2 class="h-5 w-5" /></div>
        <div class="min-w-0">
          <p class="text-xs font-bold uppercase tracking-[0.24em] text-background/55">{{ eyebrow || t('settings.eyebrow') }}</p>
          <h1 :class="cn('mt-2 text-3xl font-black tracking-[-0.04em] sm:text-5xl', layout === 'viewport' && 'lg:text-4xl')">{{ title }}</h1>
          <p :class="cn('mt-4 max-w-4xl text-sm leading-7 text-background/70', layout === 'viewport' && 'lg:mt-2 lg:leading-6')">{{ description }}</p>
        </div>
      </div>
    </header>

    <main
      :class="cn(
        'min-w-0 border-x border-b border-border bg-surface p-4 sm:p-6 lg:p-8',
        layout === 'viewport' && 'lg:min-h-0 lg:flex-1 lg:overflow-hidden lg:p-6'
      )"
    >
      <slot />
    </main>
  </PageShell>
</template>
