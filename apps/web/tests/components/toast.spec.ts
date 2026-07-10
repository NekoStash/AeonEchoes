import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Toast from '../../components/ui/Toast.vue'

describe('UiToast', () => {
  it('危险通知使用 alert 并可关闭', async () => {
    const user = userEvent.setup()
    const view = render(Toast, {
      props: {
        message: { id: 7, title: '保存失败', description: '请检查连接', tone: 'danger', duration: 0 }
      }
    })
    expect(screen.getByRole('alert')).toHaveTextContent('保存失败')
    await user.click(screen.getByRole('button', { name: '关闭通知' }))
    expect(view.emitted().dismiss).toContainEqual([7])
  })
})
