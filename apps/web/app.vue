<script setup lang="ts">
const { locale, t } = useI18n()

useHead(() => ({
  title: t('product.pageTitle'),
  htmlAttrs: { lang: locale.value },
  meta: [
    { name: 'description', content: t('product.description') },
    { property: 'og:title', content: t('product.pageTitle') },
    { property: 'og:description', content: t('product.description') }
  ]
}))
</script>

<template>
  <a href="#main-content" class="focus-ring fixed left-3 top-3 z-[100] -translate-y-20 bg-foreground px-4 py-2 text-sm font-semibold text-background transition-transform focus:translate-y-0">
    {{ $t('actions.skipToContent') }}
  </a>
  <NuxtRouteAnnouncer />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <UiToastViewport />
</template>
