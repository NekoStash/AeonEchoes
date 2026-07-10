import { describe, expect, it } from 'vitest'
import { createGraphExpandRequest, filterGraphEdges, filterGraphNodes, parseEntityIds } from '../../features/graph-explore/graph-view'
import type { GraphEdge, GraphNode } from '../../lib/types'

const nodes: GraphNode[] = [
  { id: 'a', label: 'Alpha', type: 'character', status: 'stable', depth: 1, timeline: 1, metadata: {} },
  { id: 'b', label: 'Beta', type: 'event', status: 'conflict', depth: 2, timeline: 4, metadata: { note: 'late' } }
]
const edges: GraphEdge[] = [
  { id: 'e1', project_id: 'project-1', source: 'a', target: 'b', label: 'knows', type: 'causes', weight: 0.5, timeline: 4 }
]

describe('graph explore query boundary', () => {
  it('服务端请求只构造 entity_ids 与 depth', () => {
    expect(parseEntityIds('a, b\na')).toEqual(['a', 'b'])
    expect(createGraphExpandRequest('project-1', 'a, b', 3)).toEqual({ project_id: 'project-1', entity_ids: ['a', 'b'], depth: 3 })
    expect(Object.keys(createGraphExpandRequest('project-1', '', 2)).sort()).toEqual(['depth', 'entity_ids', 'project_id'].sort())
  })

  it('本地筛选只处理已经返回的节点和边', () => {
    const filters = { search: '', nodeType: '', nodeStatus: '', edgeType: '', maxTimeline: 2 }
    const visibleNodes = filterGraphNodes(nodes, filters)
    expect(visibleNodes.map((node) => node.id)).toEqual(['a'])
    expect(filterGraphEdges(edges, new Set(visibleNodes.map((node) => node.id)), filters)).toEqual([])
  })

  it('非法 depth 会快速失败', () => {
    expect(() => createGraphExpandRequest('project-1', '', 0)).toThrow(/depth/i)
  })
})
