<script setup lang="ts">
import { Filter as FilterIcon, GitFork, Info, Loader2, Network, RefreshCw, SlidersHorizontal } from '@lucide/vue'
import type { Core, ElementDefinition } from 'cytoscape'
import DataCardGrid from '~/components/data/DataCardGrid.vue'
import DataCollection from '~/components/data/DataCollection.vue'
import DataEmptyState from '~/components/data/EmptyState.vue'
import DataErrorState from '~/components/data/ErrorState.vue'
import FilterBar from '~/components/data/FilterBar.vue'
import DataLoadingState from '~/components/data/LoadingState.vue'
import DataNoResultsState from '~/components/data/NoResultsState.vue'
import DataTable from '~/components/data/DataTable.vue'
import DensityToggle from '~/components/data/DensityToggle.vue'
import SearchInput from '~/components/data/SearchInput.vue'
import SortSelect from '~/components/data/SortSelect.vue'
import ViewModeToggle from '~/components/data/ViewModeToggle.vue'
import Panel from '~/components/ds/Panel.vue'
import StatusBadge from '~/components/ds/StatusBadge.vue'
import StatusStack from '~/components/ds/StatusStack.vue'
import PageHeader from '~/components/layout/PageHeader.vue'
import PageShell from '~/components/layout/PageShell.vue'
import SplitPane from '~/components/layout/SplitPane.vue'
import Toolbar from '~/components/layout/Toolbar.vue'
import type { GraphEdge, GraphNode } from '~/lib/types'

type GraphTab = 'graph' | 'nodes' | 'edges' | 'metadata'
type GraphViewMode = 'table' | 'grid'
type GraphDensity = 'compact' | 'comfortable' | 'relaxed'
type FilterableNodeType = Exclude<GraphNode['type'], 'story_start'>
type NodeSortKey = 'label:asc' | 'label:desc' | 'type:asc' | 'status:asc' | 'depth:asc' | 'depth:desc' | 'timeline:asc' | 'timeline:desc'
type EdgeSortKey = 'label:asc' | 'label:desc' | 'type:asc' | 'weight:desc' | 'weight:asc' | 'timeline:asc' | 'timeline:desc'
type StatusStackItem = {
  id: string
  tone?: 'info' | 'success' | 'warning' | 'danger' | 'neutral'
  title?: string
  description?: string
}

const route = useRoute()
const { t } = useI18n()
const projectId = computed(() => String(route.params.id))
const workspace = useWorkspaceStore()
const api = useApi()

const filterableNodeTypes: FilterableNodeType[] = ['character', 'location', 'event', 'clue', 'rule', 'chapter']
const knownNodeStatuses: GraphNode['status'][] = ['stable', 'draft', 'conflict', 'resolved']
const knownEdgeTypes = ['causes', 'reveals', 'depends_on', 'appears_in', 'contradicts', 'foreshadows']

const root = ref('story_start')
const timeline = ref(4)
const depth = ref(2)
const filters = ref<FilterableNodeType[]>([...filterableNodeTypes])
const graph = ref(workspace.activeGraph)
const selectedNode = ref<GraphNode | null>(null)
const selectedEdge = ref<GraphEdge | null>(null)
const loading = ref(false)
const localError = ref('')
const cytoscapeError = ref('')
const activeGraphTab = ref<GraphTab>('graph')
const nodeSearchQuery = ref('')
const nodeStatusFilter = ref<GraphNode['status'] | ''>('')
const edgeSearchQuery = ref('')
const edgeTypeFilter = ref('')
const nodeSortKey = ref<NodeSortKey>('timeline:asc')
const edgeSortKey = ref<EdgeSortKey>('timeline:asc')
const nodeViewMode = ref<GraphViewMode>('table')
const edgeViewMode = ref<GraphViewMode>('table')
const collectionDensity = ref<GraphDensity>('comfortable')
const graphContainer = ref<HTMLElement | null>(null)
const detailsPanel = ref<HTMLElement | { $el?: HTMLElement } | null>(null)
let cy: Core | null = null

const allNodes = computed<GraphNode[]>(() => graph.value?.nodes || [])
const allEdges = computed<GraphEdge[]>(() => graph.value?.edges || [])

const graphTabs = computed(() => [
  { label: t('graph.tabs.graph'), value: 'graph', badge: String(visibleNodes.value.length) },
  { label: t('graph.tabs.nodes'), value: 'nodes', badge: String(visibleNodes.value.length) },
  { label: t('graph.tabs.edges'), value: 'edges', badge: String(filteredEdges.value.length) },
  { label: t('graph.tabs.metadata'), value: 'metadata' }
])

const nodeTypeOptions = computed(() => filterableNodeTypes.map((value) => ({ label: nodeTypeLabel(value), value })))
const nodeStatusOptions = computed(() => [
  { label: t('graph.filterControls.allStatuses'), value: '' },
  ...uniqueTokens([...knownNodeStatuses, ...allNodes.value.map((node) => node.status)]).map((value) => ({
    label: nodeStatusLabel(value),
    value
  }))
])
const edgeTypeOptions = computed(() => [
  { label: t('graph.filterControls.allEdgeTypes'), value: '' },
  ...uniqueTokens([...knownEdgeTypes, ...allEdges.value.map((edge) => edge.type)]).map((value) => ({
    label: edgeTypeLabel(value),
    value
  }))
])

const nodeSortOptions = computed(() => [
  { label: t('graph.sort.nodeLabelAsc'), value: 'label:asc' },
  { label: t('graph.sort.nodeLabelDesc'), value: 'label:desc' },
  { label: t('graph.sort.nodeTypeAsc'), value: 'type:asc' },
  { label: t('graph.sort.nodeStatusAsc'), value: 'status:asc' },
  { label: t('graph.sort.nodeDepthAsc'), value: 'depth:asc' },
  { label: t('graph.sort.nodeDepthDesc'), value: 'depth:desc' },
  { label: t('graph.sort.nodeTimelineAsc'), value: 'timeline:asc' },
  { label: t('graph.sort.nodeTimelineDesc'), value: 'timeline:desc' }
])
const edgeSortOptions = computed(() => [
  { label: t('graph.sort.edgeLabelAsc'), value: 'label:asc' },
  { label: t('graph.sort.edgeLabelDesc'), value: 'label:desc' },
  { label: t('graph.sort.edgeTypeAsc'), value: 'type:asc' },
  { label: t('graph.sort.edgeWeightDesc'), value: 'weight:desc' },
  { label: t('graph.sort.edgeWeightAsc'), value: 'weight:asc' },
  { label: t('graph.sort.edgeTimelineAsc'), value: 'timeline:asc' },
  { label: t('graph.sort.edgeTimelineDesc'), value: 'timeline:desc' }
])

const nodeTableColumns = computed(() => [
  { key: 'node', label: t('graph.table.node'), class: 'min-w-[260px]', headerClass: 'min-w-[260px]' },
  { key: 'type', label: t('graph.table.type'), class: 'min-w-[120px]' },
  { key: 'status', label: t('graph.table.status'), class: 'min-w-[130px]' },
  { key: 'depth', label: t('graph.table.depth'), align: 'right' as const, class: 'min-w-[90px] tabular-nums' },
  { key: 'timeline', label: t('graph.table.timeline'), align: 'right' as const, class: 'min-w-[110px] tabular-nums' },
  { key: 'actions', label: t('graph.table.actions'), align: 'right' as const, class: 'min-w-[120px]' }
])
const edgeTableColumns = computed(() => [
  { key: 'edge', label: t('graph.table.edge'), class: 'min-w-[250px]', headerClass: 'min-w-[250px]' },
  { key: 'type', label: t('graph.table.type'), class: 'min-w-[130px]' },
  { key: 'relation', label: t('graph.table.relation'), class: 'min-w-[260px]' },
  { key: 'weight', label: t('graph.table.weight'), align: 'right' as const, class: 'min-w-[100px] tabular-nums' },
  { key: 'timeline', label: t('graph.table.timeline'), align: 'right' as const, class: 'min-w-[110px] tabular-nums' },
  { key: 'actions', label: t('graph.table.actions'), align: 'right' as const, class: 'min-w-[120px]' }
])

const scopedNodes = computed(() => allNodes.value.filter((node) => {
  if (node.depth > depth.value || node.timeline > timeline.value) return false
  return !isFilterableNodeType(node.type) || filters.value.includes(node.type)
}))
const visibleNodes = computed(() => scopedNodes.value
  .filter(nodeMatchesFilters)
  .sort(compareNodesForCollection))
const visibleNodeIds = computed(() => new Set(visibleNodes.value.map((node) => node.id)))
const visibleEdges = computed(() => allEdges.value.filter(
  (edge) =>
    edge.timeline <= timeline.value && visibleNodeIds.value.has(edge.source) && visibleNodeIds.value.has(edge.target)
))
const filteredEdges = computed(() => visibleEdges.value
  .filter(edgeMatchesFilters)
  .sort(compareEdgesForCollection))

const nodeRows = computed<Array<Record<string, unknown>>>(() => visibleNodes.value.map((node) => node as unknown as Record<string, unknown>))
const edgeRows = computed<Array<Record<string, unknown>>>(() => filteredEdges.value.map((edge) => edge as unknown as Record<string, unknown>))
const collectionTableDensity = computed<'compact' | 'comfortable' | 'relaxed'>(() => collectionDensity.value)
const panelPadding = computed<'sm' | 'md' | 'lg'>(() => {
  if (collectionDensity.value === 'compact') return 'sm'
  if (collectionDensity.value === 'relaxed') return 'lg'
  return 'md'
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

const activeGraphFilterCount = computed(() => {
  let count = 0
  if (nodeSearchQuery.value.trim()) count += 1
  if (nodeStatusFilter.value) count += 1
  if (filters.value.length !== filterableNodeTypes.length) count += 1
  return count
})
const activeNodeFilterCount = computed(() => [nodeSearchQuery.value.trim(), nodeStatusFilter.value].filter(Boolean).length)
const activeEdgeFilterCount = computed(() => [edgeSearchQuery.value.trim(), edgeTypeFilter.value].filter(Boolean).length)
const graphStatusItems = computed<StatusStackItem[]>(() => {
  const items: StatusStackItem[] = []
  if (localError.value) {
    items.push({ id: 'graph-load-error', tone: 'danger', title: t('graph.states.loadErrorTitle'), description: localError.value })
  }
  if (cytoscapeError.value) {
    items.push({ id: 'graph-canvas-warning', tone: 'warning', title: t('graph.states.canvasFallbackTitle'), description: cytoscapeError.value })
  }
  return items
})
const graphCanvasLoading = computed(() => loading.value && allNodes.value.length === 0)
const graphCanvasEmpty = computed(() => !graphCanvasLoading.value && !localError.value && allNodes.value.length === 0)
const graphCanvasNoResults = computed(() => !graphCanvasLoading.value && !localError.value && allNodes.value.length > 0 && visibleNodes.value.length === 0)
const nodeCollectionLoading = computed(() => loading.value && allNodes.value.length === 0)
const nodeCollectionError = computed(() => !nodeCollectionLoading.value && allNodes.value.length === 0 ? localError.value : '')
const nodeCollectionEmpty = computed(() => !nodeCollectionLoading.value && !nodeCollectionError.value && allNodes.value.length === 0)
const nodeCollectionNoResults = computed(() => !nodeCollectionLoading.value && !nodeCollectionError.value && allNodes.value.length > 0 && visibleNodes.value.length === 0)
const edgeCollectionLoading = computed(() => loading.value && allEdges.value.length === 0)
const edgeCollectionError = computed(() => !edgeCollectionLoading.value && allEdges.value.length === 0 && allNodes.value.length === 0 ? localError.value : '')
const edgeCollectionEmpty = computed(() => !edgeCollectionLoading.value && !edgeCollectionError.value && allEdges.value.length === 0)
const edgeCollectionNoResults = computed(() => !edgeCollectionLoading.value && !edgeCollectionError.value && allEdges.value.length > 0 && filteredEdges.value.length === 0)
const nodeResultSummary = computed(() => t('graph.filterControls.resultSummary', { visible: visibleNodes.value.length, total: allNodes.value.length }))
const edgeResultSummary = computed(() => t('graph.filterControls.resultSummary', { visible: filteredEdges.value.length, total: allEdges.value.length }))
const selectedSummary = computed(() => {
  if (selectedNode.value) return t('graph.selectedNodeSummary', { label: selectedNode.value.label })
  if (selectedEdge.value) return t('graph.selectedEdgeSummary', { label: selectedEdge.value.label })
  return t('graph.emptySelectionSummary')
})

onMounted(async () => {
  await loadGraph()
  await renderCytoscape()
})

watch([visibleNodes, visibleEdges], () => {
  void renderCytoscape()
})

watch(activeGraphTab, async (tab) => {
  if (tab !== 'graph') return
  await nextTick()
  await renderCytoscape()
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

async function revealDetailsPanel() {
  if (activeGraphTab.value !== 'graph') activeGraphTab.value = 'graph'
  await scrollToDetails()
}

async function selectNodeById(id: string) {
  selectedNode.value = visibleNodes.value.find((node) => node.id === id) || allNodes.value.find((node) => node.id === id) || null
  selectedEdge.value = null
  syncCytoscapeSelection()
  await revealDetailsPanel()
}

async function selectEdge(edge: GraphEdge) {
  selectedEdge.value = edge
  selectedNode.value = null
  syncCytoscapeSelection()
  await revealDetailsPanel()
}

async function selectEdgeById(id: string) {
  const edge = visibleEdges.value.find((item) => item.id === id) || allEdges.value.find((item) => item.id === id)
  if (!edge) return
  await selectEdge(edge)
}

async function renderCytoscape() {
  if (!import.meta.client || activeGraphTab.value !== 'graph' || !graphContainer.value) return
  try {
    const cytoscapeModule = await import('cytoscape')
    const cytoscape = cytoscapeModule.default
    if (cy && cy.container() !== graphContainer.value) {
      cy.destroy()
      cy = null
    }
    if (cy) {
      cy.elements().remove()
      cy.add(elements.value)
      cy.style(createCytoscapeStyles()).update()
    } else {
      cy = cytoscape({
        container: graphContainer.value,
        elements: elements.value,
        minZoom: 0.35,
        maxZoom: 2.2,
        style: createCytoscapeStyles(),
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

function createCytoscapeStyles() {
  const palette = createGraphPalette()
  return [
    {
      selector: 'node',
      style: {
        label: 'data(label)',
        'font-size': 11,
        color: palette.text,
        'text-outline-color': palette.textOutline,
        'text-outline-width': 2,
        'background-color': palette.nodeDefault,
        'border-color': palette.nodeBorder,
        'border-width': 1,
        width: 'mapData(depth, 0, 3, 58, 34)',
        height: 'mapData(depth, 0, 3, 58, 34)'
      }
    },
    { selector: 'node[type = "story_start"]', style: { 'background-color': palette.nodeStart } },
    { selector: 'node[type = "character"]', style: { 'background-color': palette.nodeCharacter } },
    { selector: 'node[type = "location"]', style: { 'background-color': palette.nodeLocation } },
    { selector: 'node[type = "event"]', style: { 'background-color': palette.nodeEvent } },
    { selector: 'node[type = "clue"]', style: { 'background-color': palette.nodeClue } },
    { selector: 'node[type = "rule"]', style: { 'background-color': palette.nodeRule } },
    { selector: 'node[type = "chapter"]', style: { 'background-color': palette.nodeChapter } },
    { selector: 'node[status = "conflict"]', style: { 'border-color': palette.danger, 'border-width': 3 } },
    { selector: 'node[status = "resolved"]', style: { 'border-color': palette.info, 'border-width': 2 } },
    { selector: 'node.ae-selected', style: { 'border-color': palette.selected, 'border-width': 5, 'background-color': palette.selectedMuted } },
    {
      selector: 'edge',
      style: {
        label: 'data(label)',
        'font-size': 9,
        color: palette.mutedText,
        'text-background-color': palette.textBackground,
        'text-background-opacity': 0.88,
        width: 'mapData(weight, 0, 1, 1, 4)',
        'line-color': palette.edge,
        'target-arrow-color': palette.edgeArrow,
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier'
      }
    },
    { selector: 'edge[type = "contradicts"]', style: { 'line-color': palette.danger, 'target-arrow-color': palette.danger, 'line-style': 'dashed' } },
    { selector: 'edge[type = "foreshadows"]', style: { 'line-color': palette.warning, 'target-arrow-color': palette.warning } },
    { selector: 'edge.ae-selected', style: { 'line-color': palette.selected, 'target-arrow-color': palette.selected, width: 5 } }
  ]
}

function createGraphPalette() {
  return {
    text: cssTokenColor('--foreground'),
    mutedText: cssTokenColor('--muted-foreground'),
    textOutline: cssTokenColor('--background'),
    textBackground: cssTokenColor('--surface', 0.92),
    nodeDefault: cssTokenColor('--muted-foreground', 0.86),
    nodeBorder: cssTokenColor('--border'),
    nodeStart: cssTokenColor('--primary'),
    nodeCharacter: cssTokenColor('--muted-foreground'),
    nodeLocation: cssTokenColor('--state-info'),
    nodeEvent: cssTokenColor('--state-danger'),
    nodeClue: cssTokenColor('--state-warning'),
    nodeRule: cssTokenColor('--state-success'),
    nodeChapter: cssTokenColor('--primary', 0.82),
    edge: cssTokenColor('--muted-foreground', 0.45),
    edgeArrow: cssTokenColor('--muted-foreground', 0.72),
    selected: cssTokenColor('--primary'),
    selectedMuted: cssTokenColor('--primary', 0.82),
    danger: cssTokenColor('--state-danger'),
    warning: cssTokenColor('--state-warning'),
    info: cssTokenColor('--state-info')
  }
}

function cssTokenColor(tokenName: string, alpha = 1) {
  if (!import.meta.client) return fallbackHsl(alpha)
  const raw = getComputedStyle(document.documentElement).getPropertyValue(tokenName).trim()
  const match = raw.match(/^([\d.]+)\s+([\d.]+%)\s+([\d.]+%)$/)
  if (!match) {
    console.warn(`[graph] Missing or invalid CSS color token: ${tokenName}`)
    return fallbackHsl(alpha)
  }
  return `hsla(${match[1]}, ${match[2]}, ${match[3]}, ${alpha})`
}

function fallbackHsl(alpha = 1) {
  return `hsla(215, 16%, 47%, ${alpha})`
}

function toggleFilter(value: FilterableNodeType) {
  if (filters.value.includes(value)) {
    filters.value = filters.value.filter((item) => item !== value)
  } else {
    filters.value = [...filters.value, value]
  }
}

function clearGraphFilters() {
  nodeSearchQuery.value = ''
  nodeStatusFilter.value = ''
  filters.value = [...filterableNodeTypes]
}

function clearNodeFilters() {
  nodeSearchQuery.value = ''
  nodeStatusFilter.value = ''
}

function clearEdgeFilters() {
  edgeSearchQuery.value = ''
  edgeTypeFilter.value = ''
}

function isFilterableNodeType(type: GraphNode['type']): type is FilterableNodeType {
  return type !== 'story_start'
}

function nodeMatchesFilters(node: GraphNode) {
  if (nodeStatusFilter.value && node.status !== nodeStatusFilter.value) return false
  return matchesQuery(nodeSearchQuery.value, nodeSearchFields(node))
}

function edgeMatchesFilters(edge: GraphEdge) {
  if (edgeTypeFilter.value && edge.type !== edgeTypeFilter.value) return false
  return matchesQuery(edgeSearchQuery.value, edgeSearchFields(edge))
}

function nodeSearchFields(node: GraphNode) {
  return [
    node.id,
    node.label,
    node.type,
    nodeTypeLabel(node.type),
    node.status,
    nodeStatusLabel(node.status),
    node.depth,
    node.timeline,
    JSON.stringify(node.metadata || {})
  ]
}

function edgeSearchFields(edge: GraphEdge) {
  return [
    edge.id,
    edge.label,
    edge.type,
    edgeTypeLabel(edge.type),
    edge.source,
    edge.target,
    edge.weight,
    edge.timeline,
    ...(edge.evidence_fact_ids || []),
    JSON.stringify(edge.metadata || {})
  ]
}

function matchesQuery(query: string, fields: unknown[]) {
  const normalizedQuery = normalizeSearch(query)
  if (!normalizedQuery) return true
  return fields.some((field) => normalizeSearch(field).includes(normalizedQuery))
}

function normalizeSearch(value: unknown) {
  return String(value || '').trim().toLowerCase()
}

function uniqueTokens(values: Array<string | undefined>) {
  return Array.from(new Set(values.map((value) => String(value || '').trim()).filter(Boolean)))
    .sort((left, right) => compareText(left, right))
}

function compareNodesForCollection(left: GraphNode, right: GraphNode) {
  const [field, direction] = nodeSortKey.value.split(':') as ['label' | 'type' | 'status' | 'depth' | 'timeline', 'asc' | 'desc']
  const multiplier = direction === 'asc' ? 1 : -1
  if (field === 'type') return compareText(nodeTypeLabel(left.type), nodeTypeLabel(right.type)) * multiplier || compareText(left.label, right.label)
  if (field === 'status') return compareText(nodeStatusLabel(left.status), nodeStatusLabel(right.status)) * multiplier || compareText(left.label, right.label)
  if (field === 'depth') return (left.depth - right.depth) * multiplier || compareText(left.label, right.label)
  if (field === 'timeline') return (left.timeline - right.timeline) * multiplier || compareText(left.label, right.label)
  return compareText(left.label, right.label) * multiplier || compareText(left.id, right.id)
}

function compareEdgesForCollection(left: GraphEdge, right: GraphEdge) {
  const [field, direction] = edgeSortKey.value.split(':') as ['label' | 'type' | 'weight' | 'timeline', 'asc' | 'desc']
  const multiplier = direction === 'asc' ? 1 : -1
  if (field === 'type') return compareText(edgeTypeLabel(left.type), edgeTypeLabel(right.type)) * multiplier || compareText(left.label, right.label)
  if (field === 'weight') return (left.weight - right.weight) * multiplier || compareText(left.label, right.label)
  if (field === 'timeline') return (left.timeline - right.timeline) * multiplier || compareText(left.label, right.label)
  return compareText(left.label, right.label) * multiplier || compareText(left.id, right.id)
}

function compareText(left: string, right: string) {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' })
}

function nodeFromRow(row: unknown) {
  return row as GraphNode
}

function edgeFromRow(row: unknown) {
  return row as GraphEdge
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
  return value === key ? prettifyToken(fallback) : value
}

function prettifyToken(value: string) {
  if (!value) return t('common.emptyValue')
  return value
    .split(/[-_]/g)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(' ')
}

function nodeStatusTone(status: string) {
  if (status === 'stable') return 'success' as const
  if (status === 'conflict') return 'danger' as const
  if (status === 'resolved') return 'info' as const
  if (status === 'draft') return 'warning' as const
  return 'neutral' as const
}
</script>

<template>
  <PageShell density="normal">
    <PageHeader :eyebrow="t('graph.eyebrow')" :title="t('graph.title')" :description="t('graph.description')">
      <template #actions>
        <UiButton variant="outline" class="w-full sm:w-auto" :to="`/projects/${projectId}`">{{ t('actions.back') }}</UiButton>
        <UiButton class="w-full sm:w-auto" :disabled="loading" @click="loadGraph">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ t('graph.reload') }}
        </UiButton>
      </template>
    </PageHeader>

    <StatusStack v-if="workspace.errors.length">
      <StatusAlert :errors="workspace.errors" />
    </StatusStack>
    <StatusStack v-if="graphStatusItems.length" :items="graphStatusItems" />

    <Toolbar density="compact" class="w-full">
      <template #start>
        <StatusBadge :tone="loading ? 'info' : 'neutral'" :pulse="loading">
          {{ loading ? t('graph.states.loadingTitle') : t('graph.counts', { nodes: visibleNodes.length, edges: visibleEdges.length }) }}
        </StatusBadge>
        <StatusBadge tone="muted">{{ t('graph.rootValue', { value: root || t('common.emptyValue') }) }}</StatusBadge>
        <StatusBadge tone="muted">{{ t('graph.timelineLimit', { value: timeline }) }}</StatusBadge>
        <StatusBadge tone="muted">{{ t('graph.depthLimit', { value: depth }) }}</StatusBadge>
      </template>
      <template #end>
        <UiButton size="sm" variant="outline" :disabled="loading" @click="loadGraph">
          <RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />
          {{ t('actions.refresh') }}
        </UiButton>
      </template>
    </Toolbar>

    <UiTabs v-model="activeGraphTab" :tabs="graphTabs" class="w-full" />

    <section v-if="activeGraphTab === 'graph'" class="space-y-4">
      <SplitPane ratio="sidebar" sticky-aside aside-class="space-y-4" main-class="space-y-4">
        <template #aside>
          <Panel tone="elevated" :padding="panelPadding">
            <div class="flex items-start gap-3">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-border bg-surface-muted text-muted-foreground">
                <SlidersHorizontal class="h-5 w-5" />
              </div>
              <div class="min-w-0">
                <h2 class="font-semibold">{{ t('graph.controls') }}</h2>
                <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t('graph.controlsDescription') }}</p>
              </div>
            </div>

            <div class="mt-5 space-y-5">
              <label class="block space-y-2">
                <span class="field-label">{{ t('graph.root') }}</span>
                <UiInput v-model="root" :placeholder="t('graph.placeholders.root')" />
              </label>

              <div class="grid gap-4">
                <label class="block space-y-2">
                  <span class="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                    <span class="font-medium">{{ t('graph.timeline') }}</span>
                    <StatusBadge tone="muted">{{ t('graph.timelineValue', { value: timeline }) }}</StatusBadge>
                  </span>
                  <input v-model.number="timeline" type="range" min="0" max="8" class="w-full accent-primary" />
                  <span class="block text-xs leading-5 text-muted-foreground">{{ t('graph.timelineControlHint') }}</span>
                </label>

                <label class="block space-y-2">
                  <span class="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                    <span class="font-medium">{{ t('graph.depth') }}</span>
                    <StatusBadge tone="muted">{{ t('graph.depthValue', { value: depth }) }}</StatusBadge>
                  </span>
                  <input v-model.number="depth" type="range" min="1" max="4" class="w-full accent-primary" />
                  <span class="block text-xs leading-5 text-muted-foreground">{{ t('graph.depthControlHint') }}</span>
                </label>
              </div>

              <div class="space-y-3">
                <div class="flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <FilterIcon class="h-4 w-4" />
                    {{ t('graph.filterControls.includedTypes') }}
                  </div>
                  <StatusBadge tone="muted">{{ t('graph.filterControls.selectedTypeCount', { count: filters.length }) }}</StatusBadge>
                </div>
                <div class="grid gap-2">
                  <button
                    v-for="option in nodeTypeOptions"
                    :key="option.value"
                    type="button"
                    :aria-pressed="filters.includes(option.value)"
                    :class="[
                      'focus-ring flex min-w-0 items-center justify-between gap-3 rounded-xl border px-3 py-2 text-left text-sm transition-all',
                      filters.includes(option.value) ? 'border-primary/35 bg-primary/10 text-foreground' : 'border-border bg-muted/35 text-muted-foreground'
                    ]"
                    @click="toggleFilter(option.value)"
                  >
                    <span class="truncate">{{ option.label }}</span>
                    <span class="max-w-[8rem] truncate font-mono text-xs text-muted-foreground" :title="option.value">{{ option.value }}</span>
                  </button>
                </div>
              </div>

              <div class="space-y-3">
                <p class="text-sm font-medium text-muted-foreground">{{ t('graph.searchAndFilters') }}</p>
                <SearchInput v-model="nodeSearchQuery" :label="t('graph.search.nodesLabel')" :placeholder="t('graph.search.nodes')" />
                <UiSelect v-model="nodeStatusFilter" :options="nodeStatusOptions" class="w-full" />
                <div class="flex flex-wrap gap-2">
                  <UiButton v-if="activeGraphFilterCount" size="sm" variant="outline" @click="clearGraphFilters">{{ t('graph.filterControls.clear') }}</UiButton>
                  <UiButton size="sm" :disabled="loading" @click="loadGraph">
                    <RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />
                    {{ t('graph.reload') }}
                  </UiButton>
                </div>
              </div>
            </div>
          </Panel>
        </template>

        <SplitPane ratio="detail" sticky-aside aside-class="space-y-4" main-class="min-w-0">
          <Panel tone="elevated" padding="none" class="overflow-hidden">
            <template #header>
              <div class="flex min-w-0 flex-col gap-3 border-b border-border bg-surface-muted p-4 lg:flex-row lg:items-center lg:justify-between">
                <div class="flex min-w-0 items-center gap-3">
                  <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-border bg-surface text-muted-foreground">
                    <Network class="h-5 w-5" />
                  </div>
                  <div class="min-w-0">
                    <h2 class="truncate font-semibold">{{ t('graph.canvasTitle') }}</h2>
                    <p class="truncate text-xs text-muted-foreground">{{ t('graph.counts', { nodes: visibleNodes.length, edges: visibleEdges.length }) }}</p>
                  </div>
                </div>
                <div class="flex min-w-0 flex-wrap gap-2">
                  <StatusBadge tone="muted" class="max-w-full sm:max-w-[18rem]">{{ t('graph.rootValue', { value: root || t('common.emptyValue') }) }}</StatusBadge>
                  <StatusBadge v-if="nodeStatusFilter" :tone="nodeStatusTone(nodeStatusFilter)">{{ nodeStatusLabel(nodeStatusFilter) }}</StatusBadge>
                  <StatusBadge v-if="nodeSearchQuery" tone="info">{{ t('graph.search.queryBadge', { query: nodeSearchQuery }) }}</StatusBadge>
                </div>
              </div>
            </template>

            <div class="relative min-h-[520px] bg-surface-sunken sm:min-h-[620px] 2xl:min-h-[760px]">
              <div ref="graphContainer" class="relative z-10 h-[520px] w-full sm:h-[620px] 2xl:h-[760px]" />

              <div v-if="graphCanvasLoading" class="absolute inset-0 z-30 flex items-center justify-center bg-background/85 p-6 backdrop-blur-sm">
                <DataLoadingState :title="t('graph.states.canvasLoadingTitle')" :description="t('graph.states.canvasLoadingDescription')" />
              </div>
              <div v-else-if="localError && allNodes.length === 0" class="absolute inset-0 z-30 flex items-center justify-center bg-background/90 p-6 backdrop-blur-sm">
                <DataErrorState :title="t('graph.states.loadErrorTitle')" :description="localError" />
              </div>
              <div v-else-if="graphCanvasEmpty" class="absolute inset-0 z-30 flex items-center justify-center bg-background/90 p-6 backdrop-blur-sm">
                <DataEmptyState :title="t('graph.states.canvasEmptyTitle')" :description="t('graph.states.canvasEmptyDescription')">
                  <template #actions>
                    <UiButton :disabled="loading" @click="loadGraph"><RefreshCw class="h-4 w-4" />{{ t('graph.reload') }}</UiButton>
                  </template>
                </DataEmptyState>
              </div>
              <div v-else-if="graphCanvasNoResults" class="absolute inset-0 z-30 flex items-center justify-center bg-background/90 p-6 backdrop-blur-sm">
                <DataNoResultsState :title="t('graph.states.canvasNoResultsTitle')" :description="t('graph.states.canvasNoResultsDescription')">
                  <template #actions>
                    <UiButton variant="outline" @click="clearGraphFilters">{{ t('graph.filterControls.clear') }}</UiButton>
                  </template>
                </DataNoResultsState>
              </div>
              <div v-else-if="cytoscapeError" class="absolute inset-0 z-20 overflow-auto bg-background/95 p-5 backdrop-blur-sm subtle-scrollbar">
                <UiAlert tone="warning" :title="t('graph.states.canvasFallbackTitle')" :description="t('graph.listUnavailableMessage')" />
                <DataCardGrid :items="nodeRows" class="mt-5" :density="collectionTableDensity" columns="two">
                  <template #default="{ item }">
                    <Panel as="button" type="button" tone="default" :padding="panelPadding" interactive class="w-full text-left" @click="selectNodeById(nodeFromRow(item).id)">
                      <div class="flex min-w-0 items-start justify-between gap-3">
                        <div class="min-w-0">
                          <p class="break-words font-medium" :title="nodeFromRow(item).label">{{ nodeFromRow(item).label }}</p>
                          <p class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ nodeFromRow(item).id }}</p>
                        </div>
                        <UiBadge tone="muted" class="shrink-0">{{ nodeTypeLabel(nodeFromRow(item).type) }}</UiBadge>
                      </div>
                      <div class="mt-3 flex flex-wrap gap-2">
                        <StatusBadge :tone="nodeStatusTone(nodeFromRow(item).status)">{{ nodeStatusLabel(nodeFromRow(item).status) }}</StatusBadge>
                        <UiBadge tone="muted">{{ t('graph.nodeSummary', { id: nodeFromRow(item).id, depth: nodeFromRow(item).depth, timeline: nodeFromRow(item).timeline }) }}</UiBadge>
                      </div>
                    </Panel>
                  </template>
                </DataCardGrid>
              </div>
            </div>
          </Panel>

          <template #aside>
            <Panel ref="detailsPanel" tabindex="-1" tone="elevated" :padding="panelPadding" class="outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background">
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-border bg-surface-muted text-muted-foreground">
                  <Info class="h-5 w-5" />
                </div>
                <div class="min-w-0">
                  <h2 class="font-semibold">{{ t('graph.details') }}</h2>
                  <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ selectedSummary }}</p>
                </div>
              </div>

              <div v-if="selectedNode" class="mt-5 space-y-4">
                <div class="rounded-2xl border border-border bg-muted/35 p-4">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.node') }}</p>
                  <h3 class="mt-2 break-words text-xl font-semibold">{{ selectedNode.label }}</h3>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <UiBadge tone="muted">{{ nodeTypeLabel(selectedNode.type) }}</UiBadge>
                    <StatusBadge :tone="nodeStatusTone(selectedNode.status)">{{ nodeStatusLabel(selectedNode.status) }}</StatusBadge>
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
                    <UiBadge tone="muted">{{ edgeTypeLabel(selectedEdge.type) }}</UiBadge>
                    <StatusBadge tone="warning">{{ t('graph.weightValue', { value: selectedEdge.weight }) }}</StatusBadge>
                  </div>
                </div>
                <div class="min-w-0 rounded-2xl border border-border bg-muted/35 p-4 text-sm">
                  <p class="truncate" :title="selectedEdge.source"><span class="text-muted-foreground">{{ t('graph.source') }}:</span> <span class="font-mono text-xs">{{ selectedEdge.source }}</span></p>
                  <p class="mt-2 truncate" :title="selectedEdge.target"><span class="text-muted-foreground">{{ t('graph.target') }}:</span> <span class="font-mono text-xs">{{ selectedEdge.target }}</span></p>
                  <p class="mt-2"><span class="text-muted-foreground">{{ t('graph.timeline') }}:</span> {{ t('graph.timelineValue', { value: selectedEdge.timeline }) }}</p>
                </div>
              </div>

              <DataEmptyState v-else class="mt-5" :title="t('graph.states.detailsEmptyTitle')" :description="t('graph.emptyDetails')" />

              <div class="mt-6">
                <div class="flex items-center justify-between gap-3">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.visibleEdges') }}</p>
                  <StatusBadge tone="muted">{{ visibleEdges.length }}</StatusBadge>
                </div>
                <DataEmptyState v-if="visibleEdges.length === 0" class="mt-3" :title="t('graph.states.edgesEmptyTitle')" :description="t('graph.emptyEdges')" />
                <div v-else class="mt-3 max-h-80 space-y-2 overflow-auto subtle-scrollbar">
                  <button
                    v-for="edge in visibleEdges"
                    :key="edge.id"
                    type="button"
                    :class="[
                      'focus-ring w-full min-w-0 rounded-xl border p-3 text-left text-sm transition-all hover:border-primary/35',
                      selectedEdge?.id === edge.id ? 'border-primary/45 bg-primary/10' : 'border-border bg-card'
                    ]"
                    @click="selectEdge(edge)"
                  >
                    <p class="truncate font-medium" :title="edge.label">{{ edge.label }}</p>
                    <p class="mt-1 truncate font-mono text-xs text-muted-foreground" :title="`${edge.source} → ${edge.target}`">{{ edge.source }} → {{ edge.target }}</p>
                  </button>
                </div>
              </div>
            </Panel>
          </template>
        </SplitPane>
      </SplitPane>
    </section>

    <section v-else-if="activeGraphTab === 'nodes'" class="space-y-4">
      <DataCollection
        :title="t('graph.nodeCollectionTitle')"
        :description="t('graph.nodeCollectionDescription')"
        :loading="nodeCollectionLoading"
        :error="nodeCollectionError"
        :empty="nodeCollectionEmpty"
        :no-results="nodeCollectionNoResults"
        :loading-title="t('graph.states.nodes.loadingTitle')"
        :loading-description="t('graph.states.nodes.loadingDescription')"
        :empty-title="t('graph.states.nodes.emptyTitle')"
        :empty-description="t('graph.states.nodes.emptyDescription')"
        :no-results-title="t('graph.states.nodes.noResultsTitle')"
        :no-results-description="t('graph.states.nodes.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ nodeResultSummary }}</span>
              <StatusBadge v-if="activeNodeFilterCount" tone="muted">{{ t('graph.filterControls.activeCount', { count: activeNodeFilterCount }) }}</StatusBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="nodeViewMode" :modes="['table', 'grid']" :label="t('graph.viewModeLabel')" />
              <DensityToggle v-model="collectionDensity" :densities="['compact', 'comfortable', 'relaxed']" :label="t('graph.densityLabel')" />
            </template>
          </Toolbar>
        </template>

        <template #filters>
          <FilterBar density="compact">
            <template #search>
              <SearchInput v-model="nodeSearchQuery" :label="t('graph.search.nodesLabel')" :placeholder="t('graph.search.nodes')" />
            </template>
            <UiSelect v-model="nodeStatusFilter" :options="nodeStatusOptions" class="min-w-[160px] flex-1 sm:max-w-[220px]" />
            <template #actions>
              <SortSelect v-model="nodeSortKey" :options="nodeSortOptions" class="min-w-[210px]" />
              <UiButton v-if="activeNodeFilterCount" variant="outline" @click="clearNodeFilters">{{ t('graph.filterControls.clear') }}</UiButton>
            </template>
          </FilterBar>
        </template>

        <template #no-results>
          <DataNoResultsState :title="t('graph.states.nodes.noResultsTitle')" :description="t('graph.states.nodes.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearNodeFilters">{{ t('graph.filterControls.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>

        <DataTable
          v-if="nodeViewMode === 'table'"
          :columns="nodeTableColumns"
          :rows="nodeRows"
          row-key="id"
          :density="collectionTableDensity"
          :caption="t('graph.table.nodeCaption')"
          class="hidden xl:block"
          @row-click="selectNodeById(nodeFromRow($event).id)"
        >
          <template #cell="{ row, column }">
            <div v-if="column.key === 'node'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground" :title="nodeFromRow(row).label">{{ nodeFromRow(row).label }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground" :title="nodeFromRow(row).id">{{ nodeFromRow(row).id }}</p>
            </div>
            <UiBadge v-else-if="column.key === 'type'" tone="muted">{{ nodeTypeLabel(nodeFromRow(row).type) }}</UiBadge>
            <StatusBadge v-else-if="column.key === 'status'" :tone="nodeStatusTone(nodeFromRow(row).status)">{{ nodeStatusLabel(nodeFromRow(row).status) }}</StatusBadge>
            <span v-else-if="column.key === 'depth'" class="font-mono text-sm">{{ nodeFromRow(row).depth }}</span>
            <span v-else-if="column.key === 'timeline'" class="font-mono text-sm">{{ t('graph.timelineValue', { value: nodeFromRow(row).timeline }) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex justify-end">
              <UiButton size="sm" variant="outline" @click.stop="selectNodeById(nodeFromRow(row).id)">{{ t('graph.openDetails') }}</UiButton>
            </div>
          </template>
        </DataTable>

        <DataCardGrid :items="nodeRows" :density="collectionTableDensity" columns="three" :class="nodeViewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="button" type="button" :padding="panelPadding" interactive class="w-full text-left" @click="selectNodeById(nodeFromRow(item).id)">
              <div class="flex min-w-0 items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold" :title="nodeFromRow(item).label">{{ nodeFromRow(item).label }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ nodeFromRow(item).id }}</p>
                </div>
                <StatusBadge :tone="nodeStatusTone(nodeFromRow(item).status)">{{ nodeStatusLabel(nodeFromRow(item).status) }}</StatusBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge tone="muted">{{ nodeTypeLabel(nodeFromRow(item).type) }}</UiBadge>
                <UiBadge tone="muted">{{ t('graph.depthLimit', { value: nodeFromRow(item).depth }) }}</UiBadge>
                <UiBadge tone="muted">{{ t('graph.timelineValue', { value: nodeFromRow(item).timeline }) }}</UiBadge>
              </div>
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <section v-else-if="activeGraphTab === 'edges'" class="space-y-4">
      <DataCollection
        :title="t('graph.edgeCollectionTitle')"
        :description="t('graph.edgeCollectionDescription')"
        :loading="edgeCollectionLoading"
        :error="edgeCollectionError"
        :empty="edgeCollectionEmpty"
        :no-results="edgeCollectionNoResults"
        :loading-title="t('graph.states.edges.loadingTitle')"
        :loading-description="t('graph.states.edges.loadingDescription')"
        :empty-title="t('graph.states.edges.emptyTitle')"
        :empty-description="t('graph.states.edges.emptyDescription')"
        :no-results-title="t('graph.states.edges.noResultsTitle')"
        :no-results-description="t('graph.states.edges.noResultsDescription')"
      >
        <template #toolbar>
          <Toolbar density="compact" class="w-full lg:w-auto">
            <template #start>
              <span class="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{{ edgeResultSummary }}</span>
              <StatusBadge v-if="activeEdgeFilterCount" tone="muted">{{ t('graph.filterControls.activeCount', { count: activeEdgeFilterCount }) }}</StatusBadge>
            </template>
            <template #end>
              <ViewModeToggle v-model="edgeViewMode" :modes="['table', 'grid']" :label="t('graph.viewModeLabel')" />
              <DensityToggle v-model="collectionDensity" :densities="['compact', 'comfortable', 'relaxed']" :label="t('graph.densityLabel')" />
            </template>
          </Toolbar>
        </template>

        <template #filters>
          <FilterBar density="compact">
            <template #search>
              <SearchInput v-model="edgeSearchQuery" :label="t('graph.search.edgesLabel')" :placeholder="t('graph.search.edges')" />
            </template>
            <UiSelect v-model="edgeTypeFilter" :options="edgeTypeOptions" class="min-w-[170px] flex-1 sm:max-w-[240px]" />
            <template #actions>
              <SortSelect v-model="edgeSortKey" :options="edgeSortOptions" class="min-w-[210px]" />
              <UiButton v-if="activeEdgeFilterCount" variant="outline" @click="clearEdgeFilters">{{ t('graph.filterControls.clear') }}</UiButton>
            </template>
          </FilterBar>
        </template>

        <template #no-results>
          <DataNoResultsState :title="t('graph.states.edges.noResultsTitle')" :description="t('graph.states.edges.noResultsDescription')">
            <template #actions><UiButton variant="outline" @click="clearEdgeFilters">{{ t('graph.filterControls.clear') }}</UiButton></template>
          </DataNoResultsState>
        </template>

        <DataTable
          v-if="edgeViewMode === 'table'"
          :columns="edgeTableColumns"
          :rows="edgeRows"
          row-key="id"
          :density="collectionTableDensity"
          :caption="t('graph.table.edgeCaption')"
          class="hidden xl:block"
          @row-click="selectEdge(edgeFromRow($event))"
        >
          <template #cell="{ row, column }">
            <div v-if="column.key === 'edge'" class="min-w-0 space-y-1">
              <p class="break-words font-medium text-foreground" :title="edgeFromRow(row).label">{{ edgeFromRow(row).label }}</p>
              <p class="break-all font-mono text-[11px] text-muted-foreground" :title="edgeFromRow(row).id">{{ edgeFromRow(row).id }}</p>
            </div>
            <UiBadge v-else-if="column.key === 'type'" tone="muted">{{ edgeTypeLabel(edgeFromRow(row).type) }}</UiBadge>
            <p v-else-if="column.key === 'relation'" class="break-all font-mono text-xs text-muted-foreground">{{ edgeFromRow(row).source }} → {{ edgeFromRow(row).target }}</p>
            <span v-else-if="column.key === 'weight'" class="font-mono text-sm">{{ edgeFromRow(row).weight }}</span>
            <span v-else-if="column.key === 'timeline'" class="font-mono text-sm">{{ t('graph.timelineValue', { value: edgeFromRow(row).timeline }) }}</span>
            <div v-else-if="column.key === 'actions'" class="flex justify-end">
              <UiButton size="sm" variant="outline" @click.stop="selectEdge(edgeFromRow(row))">{{ t('graph.openDetails') }}</UiButton>
            </div>
          </template>
        </DataTable>

        <DataCardGrid :items="edgeRows" :density="collectionTableDensity" columns="two" :class="edgeViewMode === 'table' ? 'xl:hidden' : ''">
          <template #default="{ item }">
            <Panel as="button" type="button" :padding="panelPadding" interactive class="w-full text-left" @click="selectEdge(edgeFromRow(item))">
              <div class="flex min-w-0 items-start justify-between gap-4">
                <div class="min-w-0 flex-1">
                  <h3 class="break-words font-semibold" :title="edgeFromRow(item).label">{{ edgeFromRow(item).label }}</h3>
                  <p class="mt-1 break-all font-mono text-[11px] text-muted-foreground">{{ edgeFromRow(item).id }}</p>
                </div>
                <UiBadge tone="muted">{{ edgeTypeLabel(edgeFromRow(item).type) }}</UiBadge>
              </div>
              <p class="mt-3 break-all font-mono text-xs text-muted-foreground">
                <GitFork class="mr-1 inline h-3.5 w-3.5" />
                {{ edgeFromRow(item).source }} → {{ edgeFromRow(item).target }}
              </p>
              <div class="mt-4 flex flex-wrap gap-2">
                <StatusBadge tone="warning">{{ t('graph.weightValue', { value: edgeFromRow(item).weight }) }}</StatusBadge>
                <UiBadge tone="muted">{{ t('graph.timelineValue', { value: edgeFromRow(item).timeline }) }}</UiBadge>
              </div>
            </Panel>
          </template>
        </DataCardGrid>
      </DataCollection>
    </section>

    <Panel v-else :padding="panelPadding" tone="elevated">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="min-w-0">
          <h2 class="text-lg font-semibold">{{ t('graph.tabs.metadata') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('graph.metadataDescription') }}</p>
        </div>
        <StatusBadge tone="muted">{{ selectedSummary }}</StatusBadge>
      </div>
      <div class="mt-5 grid gap-4 lg:grid-cols-2">
        <Panel tone="muted" padding="md">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.node') }}</p>
          <pre class="mt-3 max-h-96 overflow-auto whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground subtle-scrollbar">{{ selectedNode ? JSON.stringify(selectedNode.metadata, null, 2) : t('graph.emptyDetails') }}</pre>
        </Panel>
        <Panel tone="muted" padding="md">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('graph.edge') }}</p>
          <pre class="mt-3 max-h-96 overflow-auto whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground subtle-scrollbar">{{ selectedEdge ? JSON.stringify(selectedEdge.metadata || {}, null, 2) : t('graph.emptyDetails') }}</pre>
        </Panel>
      </div>
    </Panel>
  </PageShell>
</template>
