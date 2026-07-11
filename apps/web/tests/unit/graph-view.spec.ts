import { describe, expect, it } from 'vitest'
import { createGraphExpandRequest, filterGraphEdges, filterGraphNodes, parseEntityIds } from '../../features/graph-explore/graph-view'
import type { GraphEdge, GraphNode } from '../../lib/types'

const now = '2026-01-01T00:00:00Z'
const nodes: GraphNode[] = [
  { id: 'a', label: 'Alpha', type: 'character', status: 'stable', importance: 1, depth: 1, timeline: 1, metadata: {} },
  { id: 'b', label: 'Beta', type: 'event', status: 'conflict', importance: 0.8, depth: 2, timeline: 4, metadata: { note: 'late' } },
  { id: 'c', label: 'Unknown time', type: 'location', status: 'draft', importance: 0.4, metadata: {} }
]
const edges: GraphEdge[] = [
  { id: 'e1', project_id: 'project-1', source: 'a', target: 'b', source_entity_id: 'a', target_entity_id: 'b', label: 'knows', type: 'causes', weight: 0.5, timeline: 4, created_at: now, updated_at: now },
  { id: 'e2', project_id: 'project-1', source: 'a', target: 'c', source_entity_id: 'a', target_entity_id: 'c', label: 'visits', type: 'appears_in', weight: 0.3, created_at: now, updated_at: now }
]

describe('graph explore query boundary', () => {
  it('服务端请求只构造 entity_ids 与 depth', () => {
    expect(parseEntityIds('a, b\na')).toEqual(['a', 'b'])
    expect(createGraphExpandRequest('project-1', 'a, b', 3)).toEqual({ project_id: 'project-1', entity_ids: ['a', 'b'], depth: 3 })
    expect(Object.keys(createGraphExpandRequest('project-1', '', 2)).sort()).toEqual(['depth', 'entity_ids', 'project_id'].sort())
  })

  it('启用时间线筛选时未知 timeline 不匹配', () => {
    const filters = { search: '', nodeType: '', nodeStatus: '', edgeType: '', maxTimeline: 2 }
    const visibleNodes = filterGraphNodes(nodes, filters)
    expect(visibleNodes.map((node) => node.id)).toEqual(['a'])
    expect(filterGraphEdges(edges, new Set(visibleNodes.map((node) => node.id)), filters)).toEqual([])
  })

  it('未启用时间线筛选时未知 timeline 正常展示', () => {
    const filters = { search: '', nodeType: '', nodeStatus: '', edgeType: '', maxTimeline: null }
    const visibleNodes = filterGraphNodes(nodes, filters)
    expect(visibleNodes.map((node) => node.id)).toEqual(['a', 'b', 'c'])
    expect(filterGraphEdges(edges, new Set(visibleNodes.map((node) => node.id)), filters).map((edge) => edge.id)).toEqual(['e1', 'e2'])
  })

  it('非法 depth 会快速失败', () => {
    expect(() => createGraphExpandRequest('project-1', '', 0)).toThrow(/depth/i)
  })
})
