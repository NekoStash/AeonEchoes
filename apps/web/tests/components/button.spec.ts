import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Button from '../../components/ui/Button.vue'

describe('UiButton', () => {
  it('可通过键盘触发且保留按钮语义', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    render(Button, { attrs: { onClick }, slots: { default: '保存' } })

    const button = screen.getByRole('button', { name: '保存' })
    await user.tab()
    expect(button).toHaveFocus()
    await user.keyboard('{Enter}')
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('加载时禁用交互并声明忙碌状态', () => {
    render(Button, { props: { loading: true, loadingLabel: '正在保存' }, slots: { default: '保存' } })
    const button = screen.getByRole('button', { name: '正在保存 保存' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByText('正在保存')).toHaveClass('sr-only')
  })
})
