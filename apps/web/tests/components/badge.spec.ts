import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import Badge from '../../components/ui/Badge.vue'

describe('UiBadge', () => {
  it('运行时保持直角状态边界', () => {
    render(Badge, { props: { tone: 'success' }, slots: { default: '已启用' } })
    const badge = screen.getByText('已启用')

    expect(getComputedStyle(badge).borderRadius).toBe('0px')
  })
})
