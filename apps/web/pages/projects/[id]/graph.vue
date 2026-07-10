<script setup lang="ts">
import { AlertTriangle, Database, Filter, GitBranch, ListTree, Network, RefreshCw, Search } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import { useGraphStore } from '~/entities/graph'
import GraphCanvas from '~/features/graph-explore/GraphCanvas.vue'
import GraphDetails from '~/features/graph-explore/GraphDetails.vue'
import { createGraphExpandRequest, filterGraphEdges, filterGraphNodes, relatedEdges, type GraphListKind, type GraphPrimaryView, type GraphViewFilters } from '~/features/graph-explore/graph-view'

const route = useRoute()
const { t } = useI18n()
const toast = useToast()
const graphStore = useGraphStore()
const { expandRequest } = storeToRefs(graphStore)
const projectId = computed(() => String(route.params.id || ''))

const entityIdsInput = ref('')
const depth = ref(2)
const primaryView = ref<GraphPrimaryView>('list')
const listKind = ref<GraphListKind>('nodes')
const filtersOpen = ref(false)
const selectedNodeId = ref('')
const selectedEdgeId = ref('')
const canvasError = ref('')
const filters = reactive<GraphViewFilters>({
  search: '',
  nodeType: '',
  nodeStatus: '',
  edgeType: '',
  maxTimeline: null
})

const graph = computed(() => graphStore.byProjectId[projectId.value])
const allNodes = computed(() => graph.value?.nodes || [])
const allEdges = computed(() => graph.value?.edges || [])
const filteredNodes = computed(() => filterGraphNodes(allNodes.value, filters))
const visibleNodeIds = computed(() => new Set(filteredNodes.value.map((node) => node.id)))
const filteredEdges = computed(() => filterGraphEdges(allEdges.value, visibleNodeIds.value, filters))
const selectedNode = computed(() => allNodes.value.find((node) => node.id === selectedNodeId.value) || null)
const selectedEdge = computed(() => allEdges.value.find((edge) => edge.id === selectedEdgeId.value) || null)
const selectionId = computed(() => selectedNode.value?.id || selectedEdge.value?.id || '')
const selectionEdges = computed(() => relatedEdges(allEdges.value, selectionId.value))
const maxTimelineAvailable = computed(() => Math.max(0, ...allNodes.value.map((node) => node.timeline), ...allEdges.value.map((edge) => edge.timeline)))
const requestedEntityIds = computed(() => entityIdsInput.value.split(/[\s,，]+/u).map((item) => item.trim()).filter(Boolean))
const hasLocalFilters = computed(() => Boolean(filters.search || filters.nodeType || filters.nodeStatus || filters.edgeType || filters.maxTimeline !== null))
const nodeTypes = computed(() => uniqueTokens(allNodes.value.map((node) => node.type)))
const nodeStatuses = computed(() => uniqueTokens(allNodes.value.map((node) => node.status)))
const edgeTypes = computed(() => uniqueTokens(allEdges.value.map((edge) => edge.type)))

onMounted(() => loadGraph())

async function loadGraph() {
  canvasError.value = ''
  try {
    const request = createGraphExpandRequest(projectId.value, entityIdsInput.value, depth.value)
    await graphStore.expand(request)
    if (selectedNodeId.value && !allNodes.value.some((node) => node.id === selectedNodeId.value)) selectedNodeId.value = ''
    if (selectedEdgeId.value && !allEdges.value.some((edge) => edge.id === selectedEdgeId.value)) selectedEdgeId.value = ''
  } catch (error) {
    const message = graphStore.expandRequest.error?.message || (error instanceof Error ? error.message : t('graph.errors.loadFailed'))
    console.error('[graph-explore] Graph expansion failed.', error)
    toast.error(t('graph.states.loadErrorTitle'), message, error)
  }
}

function selectNode(id: string) {
  selectedNodeId.value = id
  selectedEdgeId.value = ''
}

function selectEdge(id: string) {
  selectedEdgeId.value = id
  selectedNodeId.value = ''
}

function clearFilters() {
  filters.search = ''
  filters.nodeType = ''
  filters.nodeStatus = ''
  filters.edgeType = ''
  filters.maxTimeline = null
}

function resetQuery() {
  entityIdsInput.value = ''
  depth.value = 2
}

function setCanvasError(message: string) {
  canvasError.value = message
}

function uniqueTokens(values: string[]) {
  return Array.from(new Set(values.filter(Boolean))).sort((left, right) => left.localeCompare(right, undefined, { sensitivity: 'base', numeric: true }))
}

function nodeTypeLabel(type: string) {
  return translatedToken(`graph.nodeTypes.${type}`, type)
}

function nodeStatusLabel(status: string) {
  return translatedToken(`status.graphNode.${status}`, status)
}

function edgeTypeLabel(type: string) {
  return translatedToken(`graph.edgeTypes.${type}`, type)
}

function translatedToken(key: string, fallback: string) {
  const value = t(key)
  return value === key ? fallback.replace(/[-_]/g, ' ') : value
}

function statusTone(status: string) {
  if (status === 'stable' || status === 'resolved') return 'success' as const
  if (status === 'conflict') return 'danger' as const
  if (status === 'draft') return 'warning' as const
  return 'muted' as const
}
</script>

<template>
  <PageShell class="pb-8">
    <header class="border-y border-border bg-foreground px-5 py-7 text-background sm:px-7 lg:px-9">
      <div class="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
        <div class="max-w-4xl">
          <p class="text-xs font-bold uppercase tracking-[0.24em] text-background/60">{{ t('graph.eyebrow') }}</p>
          <h1 class="mt-3 text-3xl font-black tracking-[-0.04em] sm:text-5xl">{{ t('graph.title') }}</h1>
          <p class="mt-4 max-w-3xl text-sm leading-7 text-background/70">{{ t('graph.description') }}</p>
        </div>
        <div class="grid grid-cols-2 border border-background/25 sm:grid-cols-4">
          <div class="border-b border-r border-background/25 px-4 py-3 sm:border-b-0"><p class="text-[10px] uppercase tracking-[0.16em] text-background/50">{{ t('graph.metrics.nodes') }}</p><p class="mt-1 text-xl font-black tabular-nums">{{ allNodes.length }}</p></div>
          <div class="border-b border-background/25 px-4 py-3 sm:border-b-0 sm:border-r"><p class="text-[10px] uppercase tracking-[0.16em] text-background/50">{{ t('graph.metrics.edges') }}</p><p class="mt-1 text-xl font-black tabular-nums">{{ allEdges.length }}</p></div>
          <div class="border-r border-background/25 px-4 py-3"><p class="text-[10px] uppercase tracking-[0.16em] text-background/50">entity_ids</p><p class="mt-1 text-xl font-black tabular-nums">{{ requestedEntityIds.length || '—' }}</p></div>
          <div class="px-4 py-3"><p class="text-[10px] uppercase tracking-[0.16em] text-background/50">depth</p><p class="mt-1 text-xl font-black tabular-nums">{{ depth }}</p></div>
        </div>
      </div>
    </header>

    <section class="grid border-x border-b border-border bg-surface lg:grid-cols-[19rem_minmax(0,1fr)]">
      <aside class="hidden border-b border-border lg:block lg:border-b-0 lg:border-r">
        <div class="border-b border-border bg-surface-muted px-5 py-4">
          <div class="flex items-center gap-3"><Database class="h-4 w-4" /><h2 class="text-sm font-black uppercase tracking-[0.14em]">{{ t('graph.query.title') }}</h2></div>
          <p class="mt-2 text-xs leading-5 text-muted-foreground">{{ t('graph.query.description') }}</p>
        </div>
        <div class="space-y-5 p-5">
          <label class="block space-y-2">
            <span class="field-label">entity_ids</span>
            <UiTextarea v-model="entityIdsInput" :rows="3" :placeholder="t('graph.query.entityIdsPlaceholder')" />
            <span class="block text-xs leading-5 text-muted-foreground">{{ t('graph.query.entityIdsHelp') }}</span>
          </label>
          <label class="block space-y-2">
            <span class="flex items-center justify-between gap-3"><span class="field-label">depth</span><strong class="font-mono text-sm">{{ depth }}</strong></span>
            <input v-model.number="depth" type="range" min="1" max="4" step="1" class="w-full accent-foreground" />
            <span class="block text-xs leading-5 text-muted-foreground">{{ t('graph.query.depthHelp') }}</span>
          </label>
          <div class="grid grid-cols-2 gap-2">
            <UiButton variant="outline" @click="resetQuery">{{ t('actions.reset') }}</UiButton>
            <UiButton :loading="expandRequest.loading" @click="loadGraph"><RefreshCw class="h-4 w-4" />{{ t('graph.query.run') }}</UiButton>
          </div>
          <div class="border border-border bg-background p-3 text-xs leading-5 text-muted-foreground">
            <p class="font-bold text-foreground">{{ t('graph.query.contractTitle') }}</p>
            <p class="mt-1">POST /projects/:projectID/graph-expansions</p>
            <p class="font-mono">{ entity_ids?, depth }</p>
          </div>
        </div>

        <div class="border-y border-border bg-surface-muted px-5 py-4">
          <div class="flex items-center gap-3"><Filter class="h-4 w-4" /><h2 class="text-sm font-black uppercase tracking-[0.14em]">{{ t('graph.localFilters.title') }}</h2></div>
          <p class="mt-2 text-xs leading-5 text-muted-foreground">{{ t('graph.localFilters.description') }}</p>
        </div>
        <div class="space-y-4 p-5">
          <label class="block space-y-2"><span class="field-label">{{ t('graph.search.nodesLabel') }}</span><div class="relative"><Search class="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" /><UiInput v-model="filters.search" class="pl-9" :placeholder="t('graph.search.nodes')" /></div></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.type') }}</span><UiSelect v-model="filters.nodeType" :options="nodeTypes.map((value) => ({ value, label: nodeTypeLabel(value) }))" :placeholder="t('graph.filterControls.allNodeTypes')" /></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.status') }}</span><UiSelect v-model="filters.nodeStatus" :options="nodeStatuses.map((value) => ({ value, label: nodeStatusLabel(value) }))" :placeholder="t('graph.filterControls.allStatuses')" /></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.relation') }}</span><UiSelect v-model="filters.edgeType" :options="edgeTypes.map((value) => ({ value, label: edgeTypeLabel(value) }))" :placeholder="t('graph.filterControls.allEdgeTypes')" /></label>
          <label v-if="maxTimelineAvailable > 0" class="block space-y-2"><span class="flex items-center justify-between gap-3"><span class="field-label">{{ t('graph.localFilters.timeline') }}</span><span class="font-mono text-xs">{{ filters.maxTimeline ?? t('graph.localFilters.allTimeline') }}</span></span><input v-model.number="filters.maxTimeline" type="range" min="0" :max="maxTimelineAvailable" class="w-full accent-foreground" /><button type="button" class="focus-ring text-xs font-bold underline underline-offset-4" @click="filters.maxTimeline = null">{{ t('graph.localFilters.clearTimeline') }}</button></label>
          <UiButton v-if="hasLocalFilters" variant="outline" class="w-full" @click="clearFilters">{{ t('graph.filterControls.clear') }}</UiButton>
        </div>
      </aside>

      <main class="min-w-0">
        <div class="flex flex-col gap-3 border-b border-border bg-background px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-wrap items-center gap-2">
            <UiButton variant="outline" class="lg:hidden" @click="filtersOpen = true"><Filter class="h-4 w-4" />{{ t('graph.localFilters.title') }}</UiButton>
            <div class="inline-flex w-fit border border-border">
            <button v-for="view in (['list', 'canvas'] as GraphPrimaryView[])" :key="view" type="button" :aria-pressed="primaryView === view" :class="['focus-ring flex min-h-10 items-center gap-2 px-4 text-sm font-black', primaryView === view ? 'bg-foreground text-background' : 'bg-background text-muted-foreground hover:bg-muted']" @click="primaryView = view">
              <ListTree v-if="view === 'list'" class="h-4 w-4" /><Network v-else class="h-4 w-4" />{{ t(`graph.views.${view}`) }}
            </button>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-xs font-semibold text-muted-foreground"><span class="border border-border px-3 py-2">{{ t('graph.localFilters.badge') }}</span><span>{{ t('graph.filterControls.resultSummary', { visible: listKind === 'nodes' ? filteredNodes.length : filteredEdges.length, total: listKind === 'nodes' ? allNodes.length : allEdges.length }) }}</span></div>
        </div>

        <div v-if="expandRequest.error && !graph" class="p-6"><UiAlert tone="danger" :title="t('graph.states.loadErrorTitle')" :description="expandRequest.error.message" /></div>
        <div v-else-if="expandRequest.loading && !graph" class="flex min-h-[30rem] items-center justify-center p-6 text-sm font-bold text-muted-foreground">{{ t('graph.states.loadingTitle') }}</div>
        <div v-else-if="!graph || allNodes.length === 0" class="flex min-h-[30rem] items-center justify-center p-6"><div class="max-w-md text-center"><GitBranch class="mx-auto h-9 w-9 text-muted-foreground" /><h2 class="mt-4 text-xl font-black">{{ t('graph.states.canvasEmptyTitle') }}</h2><p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('graph.states.canvasEmptyDescription') }}</p></div></div>

        <div v-else-if="primaryView === 'list'" class="grid min-h-[42rem] 2xl:grid-cols-[minmax(0,1fr)_22rem]">
          <section class="min-w-0 border-b border-border 2xl:border-b-0 2xl:border-r">
            <div class="flex border-b border-border">
              <button v-for="kind in (['nodes', 'edges'] as GraphListKind[])" :key="kind" type="button" :aria-pressed="listKind === kind" :class="['focus-ring flex-1 px-4 py-3 text-left text-sm font-black uppercase tracking-[0.12em]', listKind === kind ? 'bg-surface-muted text-foreground' : 'bg-background text-muted-foreground hover:bg-muted']" @click="listKind = kind">{{ t(`graph.tabs.${kind}`) }} <span class="ml-2 font-mono text-xs">{{ kind === 'nodes' ? filteredNodes.length : filteredEdges.length }}</span></button>
            </div>
            <div v-if="listKind === 'nodes'" class="divide-y divide-border">
              <button v-for="node in filteredNodes" :key="node.id" type="button" :class="['focus-ring grid w-full gap-3 px-5 py-4 text-left transition-colors sm:grid-cols-[minmax(0,1fr)_9rem_7rem_5rem]', selectedNodeId === node.id ? 'bg-foreground text-background' : 'hover:bg-surface-muted']" @click="selectNode(node.id)">
                <span class="min-w-0"><strong class="block truncate">{{ node.label || node.id }}</strong><span :class="['mt-1 block truncate font-mono text-[11px]', selectedNodeId === node.id ? 'text-background/60' : 'text-muted-foreground']">{{ node.id }}</span></span>
                <span class="text-sm font-semibold">{{ nodeTypeLabel(node.type) }}</span><span class="text-sm font-semibold">{{ nodeStatusLabel(node.status) }}</span><span class="font-mono text-sm">D{{ node.depth }} · T{{ node.timeline }}</span>
              </button>
              <div v-if="filteredNodes.length === 0" class="p-8 text-center text-sm text-muted-foreground">{{ t('graph.states.nodes.noResultsDescription') }}</div>
            </div>
            <div v-else class="divide-y divide-border">
              <button v-for="edge in filteredEdges" :key="edge.id" type="button" :class="['focus-ring grid w-full gap-3 px-5 py-4 text-left transition-colors sm:grid-cols-[minmax(0,1fr)_10rem_7rem]', selectedEdgeId === edge.id ? 'bg-foreground text-background' : 'hover:bg-surface-muted']" @click="selectEdge(edge.id)">
                <span class="min-w-0"><strong class="block truncate">{{ edge.label || edgeTypeLabel(edge.type) }}</strong><span :class="['mt-1 block truncate font-mono text-[11px]', selectedEdgeId === edge.id ? 'text-background/60' : 'text-muted-foreground']">{{ edge.source }} → {{ edge.target }}</span></span><span class="text-sm font-semibold">{{ edgeTypeLabel(edge.type) }}</span><span class="font-mono text-sm">W{{ edge.weight }} · T{{ edge.timeline }}</span>
              </button>
              <div v-if="filteredEdges.length === 0" class="p-8 text-center text-sm text-muted-foreground">{{ t('graph.states.edges.noResultsDescription') }}</div>
            </div>
          </section>
          <aside class="bg-surface-muted p-5">
            <GraphDetails :node="selectedNode" :edge="selectedEdge" :related-edges="selectionEdges" @select-edge="selectEdge" />
          </aside>
        </div>

        <div v-else class="grid min-h-[42rem] 2xl:grid-cols-[minmax(0,1fr)_22rem]">
          <section class="min-w-0 border-b border-border 2xl:border-b-0 2xl:border-r">
            <div v-if="canvasError" class="m-4 border border-state-warning-border bg-state-warning-surface p-4 text-sm text-state-warning-foreground"><div class="flex gap-3"><AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" /><div><p class="font-black">{{ t('graph.states.canvasFallbackTitle') }}</p><p class="mt-1">{{ canvasError }}</p><UiButton class="mt-3" size="sm" variant="outline" @click="primaryView = 'list'">{{ t('graph.views.openList') }}</UiButton></div></div></div>
            <GraphCanvas v-else :nodes="filteredNodes" :edges="filteredEdges" :selected-id="selectionId" @select-node="selectNode" @select-edge="selectEdge" @error="setCanvasError" />
          </section>
          <aside class="bg-surface-muted p-5"><GraphDetails :node="selectedNode" :edge="selectedEdge" :related-edges="selectionEdges" @select-edge="selectEdge" /></aside>
        </div>
      </main>
    </section>

    <UiSheet v-model:open="filtersOpen" :title="t('graph.localFilters.title')" :description="t('graph.localFilters.description')" class="w-[min(96vw,32rem)]">
      <div class="space-y-6">
        <section class="space-y-4 border-b border-border pb-6">
          <div class="flex items-center gap-3"><Database class="h-4 w-4" /><h2 class="text-sm font-black uppercase tracking-[0.14em]">{{ t('graph.query.title') }}</h2></div>
          <label class="block space-y-2"><span class="field-label">entity_ids</span><UiTextarea v-model="entityIdsInput" :rows="3" :placeholder="t('graph.query.entityIdsPlaceholder')" /></label>
          <label class="block space-y-2"><span class="flex items-center justify-between gap-3"><span class="field-label">depth</span><strong class="font-mono text-sm">{{ depth }}</strong></span><input v-model.number="depth" type="range" min="1" max="4" step="1" class="w-full accent-foreground" /></label>
          <div class="grid grid-cols-2 gap-2"><UiButton variant="outline" @click="resetQuery">{{ t('actions.reset') }}</UiButton><UiButton :loading="expandRequest.loading" @click="loadGraph"><RefreshCw class="h-4 w-4" />{{ t('graph.query.run') }}</UiButton></div>
        </section>
        <section class="space-y-4">
          <label class="block space-y-2"><span class="field-label">{{ t('graph.search.nodesLabel') }}</span><div class="relative"><Search class="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" /><UiInput v-model="filters.search" class="pl-9" :placeholder="t('graph.search.nodes')" /></div></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.type') }}</span><UiSelect v-model="filters.nodeType" :options="nodeTypes.map((value) => ({ value, label: nodeTypeLabel(value) }))" :placeholder="t('graph.filterControls.allNodeTypes')" /></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.status') }}</span><UiSelect v-model="filters.nodeStatus" :options="nodeStatuses.map((value) => ({ value, label: nodeStatusLabel(value) }))" :placeholder="t('graph.filterControls.allStatuses')" /></label>
          <label class="block space-y-2"><span class="field-label">{{ t('graph.table.relation') }}</span><UiSelect v-model="filters.edgeType" :options="edgeTypes.map((value) => ({ value, label: edgeTypeLabel(value) }))" :placeholder="t('graph.filterControls.allEdgeTypes')" /></label>
          <UiButton v-if="hasLocalFilters" variant="outline" class="w-full" @click="clearFilters">{{ t('graph.filterControls.clear') }}</UiButton>
        </section>
      </div>
    </UiSheet>
  </PageShell>
</template>
