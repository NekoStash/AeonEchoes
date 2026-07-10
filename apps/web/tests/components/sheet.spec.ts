import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Sheet from '../../components/ui/Sheet.vue'

describe('UiSheet', () => {
  it('打开时隔离页面背景并在 Escape 后恢复', async () => {
    const user = userEvent.setup()
    const background = document.createElement('main')
    background.textContent = '背景内容'
    document.body.append(background)

    const view = render(Sheet, {
      props: { open: true, title: '导航' },
      slots: { default: '<button type="button">项目</button>' }
    })
    await screen.findByRole('dialog', { name: '导航' })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(background.inert).toBe(true)
    expect(background).toHaveAttribute('aria-hidden', 'true')

    await user.keyboard('{Escape}')
    expect(view.emitted()['update:open']).toContainEqual([false])
    await view.rerender({ open: false, title: '导航' })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(background.inert).toBe(false)
    expect(background).not.toHaveAttribute('aria-hidden')
    background.remove()
  })
})
