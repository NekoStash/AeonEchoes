import { describe, expect, it } from 'vitest'
import { isRouteActive } from '../../widgets/app-navigation/navigation'

describe('isRouteActive', () => {
  it('只在根路径精确匹配工作台', () => {
    expect(isRouteActive('/', '/')).toBe(true)
    expect(isRouteActive('/projects', '/')).toBe(false)
  })

  it('匹配独立设置区子路径但不会匹配相似前缀', () => {
    expect(isRouteActive('/settings/models/detail', '/settings/models')).toBe(true)
    expect(isRouteActive('/settings/modelsmith', '/settings/models')).toBe(false)
  })

  it('支持显式精确匹配', () => {
    expect(isRouteActive('/projects/alpha/editor', '/projects/alpha', true)).toBe(false)
  })
})
