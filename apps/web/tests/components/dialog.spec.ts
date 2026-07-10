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
})
