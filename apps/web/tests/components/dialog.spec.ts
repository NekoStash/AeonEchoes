import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Dialog from '../../components/ui/Dialog.vue'

describe('UiDialog', () => {
  it('打开后聚焦首个控件，Tab 留在对话框内，Escape 关闭', async () => {
    const user = userEvent.setup()
    const view = render(Dialog, {
      props: { open: true, title: '确认操作' },
      slots: { default: '<button type="button">第一个</button><button type="button">第二个</button>' }
    })

    const dialog = await screen.findByRole('dialog', { name: '确认操作' })
    expect(getComputedStyle(dialog).borderRadius).toBe('0px')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(screen.getByRole('button', { name: '关闭' })).toHaveFocus()

    await user.keyboard('{Shift>}{Tab}{/Shift}')
    expect(screen.getByRole('button', { name: '第二个' })).toHaveFocus()

    await user.keyboard('{Escape}')
    expect(view.emitted()['update:open']).toContainEqual([false])
  })

  it('可禁用默认焦点恢复，并在完整关闭周期后发出 afterClose', async () => {
    const opener = document.createElement('button')
    opener.textContent = '外部触发器'
    document.body.appendChild(opener)
    opener.focus()
    const view = render(Dialog, {
      props: { open: true, title: '覆盖确认', restoreFocus: false }
    })
    await screen.findByRole('dialog', { name: '覆盖确认' })
    await new Promise(resolve => setTimeout(resolve, 0))

    await view.rerender({ open: false, title: '覆盖确认', restoreFocus: false })
    await new Promise(resolve => setTimeout(resolve, 0))
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(opener).not.toHaveFocus()
    expect(view.emitted('afterClose')).toHaveLength(1)
    opener.remove()
  })
})
