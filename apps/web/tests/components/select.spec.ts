import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Select from '../../components/ui/Select.vue'

describe('UiSelect', () => {
  it('转发可访问属性并通过键盘选择选项', async () => {
    const user = userEvent.setup()
    const view = render(Select, {
      attrs: { 'aria-describedby': 'provider-help' },
      props: {
        ariaLabel: '提供商',
        modelValue: '',
        options: [
          { label: 'OpenAI', value: 'openai' },
          { label: 'Anthropic', value: 'anthropic' }
        ]
      }
    })

    const trigger = screen.getByRole('button', { name: '提供商' })
    expect(trigger).toHaveAttribute('aria-describedby', 'provider-help')
    await user.click(trigger)
    await user.keyboard('{ArrowDown}{Enter}')
    expect(view.emitted()['update:modelValue']).toContainEqual(['anthropic'])
    expect(trigger).toHaveFocus()
  })

  it('搜索框具有独立可访问名称', async () => {
    const user = userEvent.setup()
    render(Select, {
      props: {
        ariaLabel: '模型',
        searchable: true,
        searchLabel: '搜索模型',
        options: [{ label: 'GPT', value: 'gpt' }]
      }
    })
    await user.click(screen.getByRole('button', { name: '模型' }))
    expect(screen.getByRole('searchbox', { name: '搜索模型' })).toHaveFocus()
  })
})
