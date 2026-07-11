import type { GraphEdge, GraphExpandRequest, GraphNode } from '~/lib/types'

export type GraphPrimaryView = 'list' | 'canvas'
export type GraphListKind = 'nodes' | 'edges'

export interface GraphViewFilters {
  search: string
  nodeType: string
  nodeStatus: string
  edgeType: string
  maxTimeline: number | null
}

export function parseEntityIds(value: string): string[] {
  return Array.from(new Set(value
    .split(/[\s,，]+/u)
    .map((item) => item.trim())
    .filter(Boolean)))
}

export function createGraphExpandRequest(projectId: string, entityIdsInput: string, depth: number): GraphExpandRequest {
  const normalizedProjectId = projectId.trim()
  if (!normalizedProjectId) {
    const error = new Error('Graph expansion requires project_id.')
    console.error('[graph-explore] Missing project_id.', error)
    throw error
  }
  if (!Number.isInteger(depth) || depth < 1 || depth > 4) {
    const error = new Error('Graph expansion depth must be an integer between 1 and 4.')
    console.error('[graph-explore] Invalid depth.', { depth, error })
    throw error
  }
  const entityIds = parseEntityIds(entityIdsInput)
  return {
    project_id: normalizedProjectId,
    depth,
    entity_ids: entityIds.length > 0 ? entityIds : undefined
  }
}

export function filterGraphNodes(nodes: GraphNode[], filters: GraphViewFilters): GraphNode[] {
  const query = normalizeSearch(filters.search)
  return nodes.filter((node) => {
    if (filters.nodeType && node.type !== filters.nodeType) return false
    if (filters.nodeStatus && node.status !== filters.nodeStatus) return false
    if (filters.maxTimeline !== null && (node.timeline === undefined || node.timeline > filters.maxTimeline)) return false
    if (!query) return true
    return [node.id, node.label, node.type, node.status, node.depth, node.timeline, JSON.stringify(node.metadata || {})]
      .some((value) => normalizeSearch(value).includes(query))
  })
}

export function filterGraphEdges(edges: GraphEdge[], visibleNodeIds: Set<string>, filters: GraphViewFilters): GraphEdge[] {
  const query = normalizeSearch(filters.search)
  return edges.filter((edge) => {
    if (!visibleNodeIds.has(edge.source) || !visibleNodeIds.has(edge.target)) return false
    if (filters.edgeType && edge.type !== filters.edgeType) return false
    if (filters.maxTimeline !== null && (edge.timeline === undefined || edge.timeline > filters.maxTimeline)) return false
    if (!query) return true
    return [edge.id, edge.label, edge.type, edge.source, edge.target, edge.weight, edge.timeline, ...(edge.evidence_fact_ids || [])]
      .some((value) => normalizeSearch(value).includes(query))
  })
}

export function relatedEdges(edges: GraphEdge[], selectionId: string): GraphEdge[] {
  if (!selectionId) return []
  return edges.filter((edge) => edge.id === selectionId || edge.source === selectionId || edge.target === selectionId)
}

function normalizeSearch(value: unknown) {
  return String(value ?? '').trim().toLocaleLowerCase()
}
