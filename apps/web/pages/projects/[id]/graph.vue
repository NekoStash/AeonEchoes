<script setup lang="ts">
import { Filter, GitFork, Info, Loader2, Network, RefreshCw, SlidersHorizontal } from '@lucide/vue'
import type { Core, ElementDefinition } from 'cytoscape'
import type { GraphEdge, GraphNode } from '~/lib/types'

const route = useRoute()
const { t } = useI18n()
const projectId = computed(() => String(route.params.id))
const workspace = useWorkspaceStore()
const api = useApi()

const root = ref('story_start')
const timeline = ref(4)
const depth = ref(2)
const filters = ref(['character', 'location', 'event', 'clue', 'rule', 'chapter'])
const graph = ref(workspace.activeGraph)
const selectedNode = ref<GraphNode | null>(null)
const selectedEdge = ref<GraphEdge | null>(null)
const loading = ref(false)
const localError = ref('')
const cytoscapeError = ref('')
const graphContainer = ref<HTMLElement | null>(null)
const detailsPanel = ref<HTMLElement | { $el?: HTMLElement } | null>(null)
let cy: Core | null = null

const nodeTypeOptions = computed(() => [
  { label: t('graph.nodeTypes.character'), value: 'character' },
  { label: t('graph.nodeTypes.location'), value: 'location' },
  { label: t('graph.nodeTypes.event'), value: 'event' },
  { label: t('graph.nodeTypes.clue'), value: 'clue' },
  { label: t('graph.nodeTypes.rule'), value: 'rule' },
  { label: t('graph.nodeTypes.chapter'), value: 'chapter' }
])

const visibleNodes = computed(() => {
  const data = graph.value?.nodes || []
  return data.filter((node) => node.depth <= depth.value && node.timeline <= timeline.value && (node.type === 'story_start' || filters.value.includes(node.type)))
})

const visibleNodeIds = computed(() => new Set(visibleNodes.value.map((node) => node.id)))
const visibleEdges = computed(() => {
  const data = graph.value?.edges || []
  return data.filter(
    (edge) =>
      edge.timeline <= timeline.value && visibleNodeIds.value.has(edge.source) && visibleNodeIds.value.has(edge.target)
  )
})

const elements = computed<ElementDefinition[]>(() => [
  ...visibleNodes.value.map((node) => ({
    data: {
      id: node.id,
      label: node.label,
      type: node.type,
      status: node.status,
      depth: node.depth
    }
  })),
  ...visibleEdges.value.map((edge) => ({
    data: {
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.label,
      type: edge.type,
      weight: edge.weight
    }
  }))
])

onMounted(async () => {
  await loadGraph()
  await renderCytoscape()
})

watch([visibleNodes, visibleEdges], () => {
  renderCytoscape()
})

onBeforeUnmount(() => {
  if (cy) {
    cy.destroy()
    cy = null
  }
})

async function loadGraph() {
  localError.value = ''
  loading.value = true
  try {
    const entityIds = root.value && root.value !== 'story_start' ? [root.value] : undefined
    const result = await api.expandGraph({
      project_id: projectId.value,
      root: root.value,
      depth: depth.value,
      timeline: timeline.value,
      filters: filters.value,
      entity_ids: entityIds
    })
    workspace.recordResult(t('graph.resultScope'), result)
    workspace.activeGraph = result.data
    graph.value = result.data
  } catch (error) {
    const apiError = workspace.recordError(t('graph.resultScope'), error)
    localError.value = apiError.message || t('graph.errors.loadFailed')
  } finally {
    loading.value = false
  }
}

function resolveElement(element: HTMLElement | { $el?: HTMLElement } | null) {
  if (!element) return null
  if (element instanceof HTMLElement) return element
  return element.$el || null
}

async function scrollToDetails() {
  await nextTick()
  const target = resolveElement(detailsPanel.value)
  target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  target?.focus({ preventScroll: true })
}

function syncCytoscapeSelection() {
  if (!cy) return
  cy.elements().removeClass('ae-selected')
  if (selectedNode.value) cy.$id(selectedNode.value.id).addClass('ae-selected')
  if (selectedEdge.value) cy.$id(selectedEdge.value.id).addClass('ae-selected')
}

async function selectNodeById(id: string) {
  selectedNode.value = visibleNodes.value.find((node) => node.id === id) || null
  selectedEdge.value = null
  syncCytoscapeSelection()
  await scrollToDetails()
}

async function selectEdge(edge: GraphEdge) {
  selectedEdge.value = edge
  selectedNode.value = null
  syncCytoscapeSelection()
  await scrollToDetails()
}

async function selectEdgeById(id: string) {
  const edge = visibleEdges.value.find((item) => item.id === id)
  if (!edge) return
  await selectEdge(edge)
}

async function renderCytoscape() {
  if (!import.meta.client || !graphContainer.value) return
  try {
    const cytoscapeModule = await import('cytoscape')
    const cytoscape = cytoscapeModule.default
    if (cy) {
      cy.elements().remove()
      cy.add(elements.value)
    } else {
      cy = cytoscape({
        container: graphContainer.value,
        elements: elements.value,
        minZoom: 0.35,
        maxZoom: 2.2,
        style: [
          {
            selector: 'node',
            style: {
              label: 'data(label)',
              'font-size': 11,
              color: '#E5E7EB',
              'text-outline-color': '#111827',
              'text-outline-width': 2,
              'background-color': '#64748B',
              'border-color': '#CBD5E1',
              'border-width': 1,
              width: 'mapData(depth, 0, 3, 58, 34)',
              height: 'mapData(depth, 0, 3, 58, 34)'
            }
          },
          {
            selector: 'node[type = "character"]',
            style: { 'background-color': '#475569' }
          },
          {
            selector: 'node[type = "event"]',
            style: { 'background-color': '#7F1D1D' }
          },
          {
            selector: 'node[type = "clue"]',
            style: { 'background-color': '#92400E' }
          },
          {
            selector: 'node[status = "conflict"]',
            style: { 'border-color': '#DC2626', 'border-width': 3 }
          },
          {
            selector: 'node.ae-selected',
            style: { 'border-color': '#38BDF8', 'border-width': 5, 'background-color': '#2563EB' }
          },
          {
            selector: 'edge',
            style: {
              label: 'data(label)',
              'font-size': 9,
              color: '#64748B',
              'text-background-color': '#F8FAFC',
              'text-background-opacity': 0.85,
              width: 'mapData(weight, 0, 1, 1, 4)',
              'line-color': 'rgba(100, 116, 139, 0.5)',
              'target-arrow-color': 'rgba(100, 116, 139, 0.75)',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier'
            }
          },
          {
            selector: 'edge[type = "contradicts"]',
            style: { 'line-color': '#DC2626', 'target-arrow-color': '#DC2626', 'line-style': 'dashed' }
          },
          {
            selector: 'edge[type = "foreshadows"]',
            style: { 'line-color': '#B45309', 'target-arrow-color': '#B45309' }
          },
          {
            selector: 'edge.ae-selected',
            style: { 'line-color': '#2563EB', 'target-arrow-color': '#2563EB', width: 5 }
          }
        ],
        layout: {
          name: 'cose',
          animate: true,
          fit: true,
          padding: 45,
          nodeRepulsion: 6000,
          idealEdgeLength: 120
        }
      })
      cy.on('tap', 'node', (event) => {
        void selectNodeById(event.target.id())
      })
      cy.on('tap', 'edge', (event) => {
        void selectEdgeById(event.target.id())
      })
      cy.on('tap', (event) => {
        if (event.target === cy) {
          selectedNode.value = null
          selectedEdge.value = null
          syncCytoscapeSelection()
        }
      })
    }
    cy.layout({ name: 'cose', animate: true, fit: true, padding: 45 }).run()
    syncCytoscapeSelection()
    cytoscapeError.value = ''
  } catch (error) {
    console.error('Cytoscape lazy load failed', error)
    cytoscapeError.value = error instanceof Error ? error.message : t('graph.errors.cytoscapeFailed')
  }
}

function toggleFilter(value: string) {
  if (filters.value.includes(value)) {
    filters.value = filters.value.filter((item) => item !== value)
  } else {
    filters.value = [...filters.value, value]
  }
}

function nodeTypeLabel(type: string) {
  return t(`graph.nodeTypes.${type}`)
}

function nodeStatusLabel(status: string) {
  return t(`status.graphNode.${status}`)
}

function edgeTypeLabel(type: string) {
  return t(`graph.edgeTypes.${type}`)
}
</script>

<template>
  <div class="space-y-6">
    <SectionHeader
      :title="t('graph.title')"
      :description="t('graph.description')"
    >
      <template #actions>
        <UiButton variant="outline" :to="`/projects/${projectId}`">{{ t('actions.back') }}</UiButton>
        <UiButton :disabled="loading" @click="loadGraph">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ t('graph.reload') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="workspace.errors" />
    <div v-if="localError || cytoscapeError" class="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      <p v-if="localError">{{ localError }}</p>
      <p v-if="cytoscapeError">{{ cytoscapeError }}</p>
    </div>

    <div class="grid min-w-0 gap-6 2xl:grid-cols-[minmax(0,320px)_minmax(0,1fr)_minmax(0,360px)]">
      <UiCard class="p-4 sm:p-5">
        <div class="flex items-center gap-2">
          <SlidersHorizontal class="h-5 w-5 text-muted-foreground" />
          <h2 class="font-semibold">{{ t('graph.controls') }}</h2>
        </div>

        <div class="mt-5 space-y-5">
          <label class="space-y-2">
            <span class="text-sm text-muted-foreground">{{ t('graph.root') }}</span>
            <UiInput v-model="root" :placeholder="t('graph.placeholders.root')" />
          </label>

          <label class="space-y-2">
            <span class="flex items-center justify-between text-sm text-muted-foreground">
              <span>{{ t('graph.timeline') }}</span>
              <UiBadge variant="muted">{{ t('graph.timelineValue', { value: timeline }) }}</UiBadge>
            </span>
            <input v-model.number="timeline" type="range" min="0" max="8" class="w-full accent-primary" />
          </label>

          <label class="space-y-2">
            <span class="flex items-center justify-between text-sm text-muted-foreground">
              <span>{{ t('graph.depth') }}</span>
              <UiBadge variant="muted">{{ depth }}</UiBadge>
            </span>
            <input v-model.number="depth" type="range" min="1" max="4" class="w-full accent-primary" />
          </label>

          <div>
            <div class="mb-3 flex items-center gap-2 text-sm text-muted-foreground">
              <Filter class="h-4 w-4" />
              {{ t('graph.filters') }}
            </div>
            <div class="grid gap-2">
              <button
                v-for="option in nodeTypeOptions"
                :key="option.value"
                type="button"
                :class="[
                  'flex min-w-0 items-center justify-between gap-3 rounded-xl border px-3 py-2 text-sm transition-all focus-ring',
                  filters.includes(option.value) ? 'border-primary/35 bg-primary/10 text-foreground' : 'border-border bg-muted/35 text-muted-foreground'
                ]"
                @click="toggleFilter(option.value)"
              >
                <span class="truncate">{{ option.label }}</span>
                <span class="max-w-[8rem] truncate font-mono text-xs text-muted-foreground" :title="option.value">{{ option.value }}</span>
              </button>
            </div>
          </div>
        </div>
      </UiCard>

      <UiCard>
        <div class="flex min-w-0 flex-col gap-3 rounded-t-2xl border-b border-border bg-muted/35 p-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border bg-card text-muted-foreground">
              <Network class="h-5 w-5" />
            </div>
            <div class="min-w-0">
              <h2 class="truncate font-semibold">{{ t('graph.title') }}</h2>
              <p class="truncate text-xs text-muted-foreground">{{ t('graph.counts', { nodes: visibleNodes.length, edges: visibleEdges.length }) }}</p>
            </div>
          </div>
          <div class="flex min-w-0 flex-wrap gap-2">
            <UiBadge variant="muted" class="max-w-full sm:max-w-[18rem]">
              <span class="truncate" :title="t('graph.rootValue', { value: root })">{{ t('graph.rootValue', { value: root }) }}</span>
            </UiBadge>
            <UiBadge variant="muted">{{ t('graph.timelineLimit', { value: timeline }) }}</UiBadge>
            <UiBadge variant="muted">{{ t('graph.depthLimit', { value: depth }) }}</UiBadge>
          </div>
        </div>

        <div class="relative min-h-[460px] rounded-b-2xl bg-muted/30 sm:min-h-[560px] 2xl:min-h-[720px]">
          <div ref="graphContainer" class="relative z-10 h-[460px] w-full sm:h-[560px] 2xl:h-[720px]" />
          <div v-if="cytoscapeError" class="absolute inset-0 z-20 overflow-auto bg-background/95 p-6 subtle-scrollbar">
            <div class="rounded-2xl border border-amber-300/40 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100">
              {{ t('graph.listUnavailableMessage') }}
            </div>
            <div class="mt-5 grid gap-4 md:grid-cols-2">
              <div v-for="node in visibleNodes" :key="node.id" class="min-w-0 rounded-2xl border border-border bg-card p-4">
                <div class="flex min-w-0 items-center justify-between gap-3">
                  <p class="truncate font-medium" :title="node.label">{{ node.label }}</p>
                  <UiBadge class="shrink-0" variant="muted">{{ nodeTypeLabel(node.type) }}</UiBadge>
                </div>
                <p class="mt-2 truncate font-mono text-xs text-muted-foreground" :title="t('graph.nodeSummary', { id: node.id, depth: node.depth, timeline: node.timeline })">{{ t('graph.nodeSummary', { id: node.id, depth: node.depth, timeline: node.timeline }) }}</p>
              </div>
            </div>
          </div>
        </div>
      </UiCard>

      <UiCard ref="detailsPanel" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-5">
        <div class="flex items-center gap-2">
          <Info class="h-5 w-5 text-muted-foreground" />
          <h2 class="font-semibold">{{ t('graph.details') }}</h2>
        </div>

        <div v-if="selectedNode" class="mt-5 space-y-4">
          <div class="rounded-2xl border border-border bg-muted/35 p-4">
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.node') }}</p>
            <h3 class="mt-2 break-words text-xl font-semibold">{{ selectedNode.label }}</h3>
            <div class="mt-3 flex flex-wrap gap-2">
              <UiBadge variant="muted">{{ nodeTypeLabel(selectedNode.type) }}</UiBadge>
              <UiBadge :variant="selectedNode.status === 'conflict' ? 'rose' : selectedNode.status === 'stable' ? 'success' : 'gold'">{{ nodeStatusLabel(selectedNode.status) }}</UiBadge>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3 text-sm">
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="text-xs text-muted-foreground">{{ t('graph.depth') }}</p>
              <p class="mt-1 font-medium">{{ selectedNode.depth }}</p>
            </div>
            <div class="rounded-xl bg-muted/35 p-3">
              <p class="text-xs text-muted-foreground">{{ t('graph.timeline') }}</p>
              <p class="mt-1 font-medium">{{ t('graph.timelineValue', { value: selectedNode.timeline }) }}</p>
            </div>
          </div>
          <div class="rounded-2xl border border-border bg-muted/35 p-4">
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.metadata') }}</p>
            <pre class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground subtle-scrollbar">{{ JSON.stringify(selectedNode.metadata, null, 2) }}</pre>
          </div>
        </div>

        <div v-else-if="selectedEdge" class="mt-5 space-y-4">
          <div class="rounded-2xl border border-border bg-muted/35 p-4">
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.edge') }}</p>
            <h3 class="mt-2 break-words text-xl font-semibold">{{ selectedEdge.label }}</h3>
            <div class="mt-3 flex flex-wrap gap-2">
              <UiBadge variant="muted">{{ edgeTypeLabel(selectedEdge.type) }}</UiBadge>
              <UiBadge variant="gold">{{ t('graph.weightValue', { value: selectedEdge.weight }) }}</UiBadge>
            </div>
          </div>
          <div class="min-w-0 rounded-2xl border border-border bg-muted/35 p-4 text-sm">
            <p class="truncate" :title="selectedEdge.source"><span class="text-muted-foreground">{{ t('graph.source') }}:</span> <span class="font-mono text-xs">{{ selectedEdge.source }}</span></p>
            <p class="mt-2 truncate" :title="selectedEdge.target"><span class="text-muted-foreground">{{ t('graph.target') }}:</span> <span class="font-mono text-xs">{{ selectedEdge.target }}</span></p>
            <p class="mt-2"><span class="text-muted-foreground">{{ t('graph.timeline') }}:</span> {{ t('graph.timelineValue', { value: selectedEdge.timeline }) }}</p>
          </div>
        </div>

        <div v-else class="mt-5 rounded-2xl border border-border bg-muted/35 p-5 text-sm leading-6 text-muted-foreground">
          {{ t('graph.emptyDetails') }}
        </div>

        <div class="mt-6">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.visibleEdges') }}</p>
          <div v-if="visibleEdges.length === 0" class="mt-3 rounded-xl border border-border bg-muted/35 p-3 text-sm text-muted-foreground">
            {{ t('graph.emptyEdges') }}
          </div>
          <div v-else class="mt-3 max-h-80 space-y-2 overflow-auto subtle-scrollbar">
            <button
              v-for="edge in visibleEdges"
              :key="edge.id"
              type="button"
              :class="[
                'w-full min-w-0 rounded-xl border p-3 text-left text-sm transition-all hover:border-primary/35 focus-ring',
                selectedEdge?.id === edge.id ? 'border-primary/45 bg-primary/10' : 'border-border bg-card'
              ]"
              @click="selectEdge(edge)"
            >
              <p class="truncate font-medium" :title="edge.label">{{ edge.label }}</p>
              <p class="mt-1 truncate font-mono text-xs text-muted-foreground" :title="`${edge.source} → ${edge.target}`">{{ edge.source }} → {{ edge.target }}</p>
            </button>
          </div>
        </div>
      </UiCard>
    </div>
  </div>
</template>
