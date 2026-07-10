<script setup lang="ts">
import { ArrowRight, Braces } from '@lucide/vue'
import type { GraphEdge, GraphNode } from '~/lib/types'

withDefaults(defineProps<{
  node?: GraphNode | null
  edge?: GraphEdge | null
  relatedEdges?: GraphEdge[]
}>(), {
  node: null,
  edge: null,
  relatedEdges: () => []
})

const emit = defineEmits<{
  selectEdge: [id: string]
}>()

const { t } = useI18n()
</script>

<template>
  <div class="space-y-5">
    <div class="border-b border-border pb-4">
      <div class="flex items-center gap-2 text-xs font-black uppercase tracking-[0.16em] text-muted-foreground"><Braces class="h-4 w-4" />{{ t('graph.details') }}</div>
      <h2 class="mt-3 break-words text-2xl font-black tracking-tight">{{ node?.label || edge?.label || t('graph.states.detailsEmptyTitle') }}</h2>
      <p class="mt-2 break-all font-mono text-[11px] text-muted-foreground">{{ node?.id || edge?.id || t('graph.emptySelectionSummary') }}</p>
    </div>

    <div v-if="node" class="space-y-3 text-sm">
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.table.type') }}</span><strong class="break-all font-mono text-xs">{{ node.type }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.table.status') }}</span><strong class="break-all font-mono text-xs">{{ node.status }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.depth') }}</span><strong class="font-mono text-xs">{{ node.depth }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.timeline') }}</span><strong class="font-mono text-xs">{{ node.timeline }}</strong></div>
      <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words border border-border bg-background p-3 text-xs leading-5">{{ JSON.stringify(node.metadata || {}, null, 2) }}</pre>
    </div>

    <div v-else-if="edge" class="space-y-3 text-sm">
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.table.type') }}</span><strong class="break-all font-mono text-xs">{{ edge.type }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.source') }}</span><strong class="break-all font-mono text-xs">{{ edge.source }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.target') }}</span><strong class="break-all font-mono text-xs">{{ edge.target }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.table.weight') }}</span><strong class="font-mono text-xs">{{ edge.weight }}</strong></div>
      <div class="grid grid-cols-[6rem_minmax(0,1fr)] gap-3 border-b border-border pb-2"><span class="text-muted-foreground">{{ t('graph.timeline') }}</span><strong class="font-mono text-xs">{{ edge.timeline }}</strong></div>
      <pre v-if="edge.metadata" class="max-h-72 overflow-auto whitespace-pre-wrap break-words border border-border bg-background p-3 text-xs leading-5">{{ JSON.stringify(edge.metadata, null, 2) }}</pre>
    </div>

    <p v-else class="text-sm leading-6 text-muted-foreground">{{ t('graph.emptyDetails') }}</p>

    <div v-if="relatedEdges.length" class="border-t border-border pt-4">
      <p class="text-xs font-black uppercase tracking-[0.16em] text-muted-foreground">{{ t('graph.visibleEdges') }}</p>
      <div class="mt-3 divide-y divide-border border-y border-border">
        <button v-for="related in relatedEdges" :key="related.id" type="button" class="focus-ring flex w-full items-center justify-between gap-3 py-3 text-left text-sm hover:text-muted-foreground" @click="emit('selectEdge', related.id)">
          <span class="min-w-0 truncate">{{ related.label || related.type }}</span><ArrowRight class="h-4 w-4 shrink-0" />
        </button>
      </div>
    </div>
  </div>
</template>
