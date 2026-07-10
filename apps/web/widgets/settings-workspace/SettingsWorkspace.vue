<script setup lang="ts">
import { Bot, Boxes, Cpu, DatabaseZap, Settings2 } from '@lucide/vue'

const props = defineProps<{
  title: string
  description: string
  eyebrow?: string
}>()

const route = useRoute()
const { t } = useI18n()
const items = computed(() => [
  { to: '/settings/providers', label: t('settings.nav.providers'), description: t('settings.navDescriptions.providers'), icon: Boxes },
  { to: '/settings/models', label: t('settings.nav.models'), description: t('settings.navDescriptions.models'), icon: Cpu },
  { to: '/settings/agents', label: t('settings.nav.agents'), description: t('settings.navDescriptions.agents'), icon: Bot },
  { to: '/settings/index-maintenance', label: t('settings.nav.indexMaintenance'), description: t('settings.navDescriptions.indexMaintenance'), icon: DatabaseZap }
])
</script>

<template>
  <PageShell class="pb-8">
    <header class="border-y border-border bg-foreground px-5 py-7 text-background sm:px-7 lg:px-9">
      <div class="flex items-start gap-4">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center border border-background/30"><Settings2 class="h-5 w-5" /></div>
        <div class="min-w-0">
          <p class="text-xs font-bold uppercase tracking-[0.24em] text-background/55">{{ eyebrow || t('settings.eyebrow') }}</p>
          <h1 class="mt-2 text-3xl font-black tracking-[-0.04em] sm:text-5xl">{{ title }}</h1>
          <p class="mt-4 max-w-4xl text-sm leading-7 text-background/70">{{ description }}</p>
        </div>
      </div>
    </header>

    <div class="grid border-x border-b border-border bg-surface lg:grid-cols-[18rem_minmax(0,1fr)]">
      <aside class="border-b border-border bg-surface-muted lg:border-b-0 lg:border-r">
        <div class="border-b border-border px-5 py-4"><p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('settings.navigation') }}</p></div>
        <nav :aria-label="t('settings.navigation')" class="divide-y divide-border">
          <NuxtLink v-for="item in items" :key="item.to" :to="item.to" :aria-current="route.path === item.to ? 'page' : undefined" :class="['focus-ring flex gap-3 px-5 py-4 transition-colors', route.path === item.to ? 'bg-foreground text-background' : 'hover:bg-background']">
            <component :is="item.icon" class="mt-0.5 h-4 w-4 shrink-0" />
            <span class="min-w-0"><strong class="block text-sm">{{ item.label }}</strong><span :class="['mt-1 block text-xs leading-5', route.path === item.to ? 'text-background/60' : 'text-muted-foreground']">{{ item.description }}</span></span>
          </NuxtLink>
        </nav>
      </aside>
      <main class="min-w-0 p-4 sm:p-6 lg:p-8"><slot /></main>
    </div>
  </PageShell>
</template>
