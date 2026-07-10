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

  it('运行时保持触发器和 Teleport 列表直角', async () => {
    const user = userEvent.setup()
    render(Select, {
      props: {
        ariaLabel: '模型',
        options: [{ label: 'GPT', value: 'gpt' }]
      }
    })

    const trigger = screen.getByRole('button', { name: '模型' })
    expect(getComputedStyle(trigger).borderRadius).toBe('0px')
    await user.click(trigger)
    const listbox = screen.getByRole('listbox', { name: '模型' })
    expect(getComputedStyle(listbox).borderRadius).toBe('0px')
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

  it('可通过键盘选择空值占位项以恢复继承状态', async () => {
    const user = userEvent.setup()
    const view = render(Select, {
      props: {
        ariaLabel: '用途模型',
        modelValue: 'gpt',
        placeholder: '继承后端路由',
        options: [{ label: 'GPT', value: 'gpt' }]
      }
    })

    await user.click(screen.getByRole('button', { name: '用途模型' }))
    await user.keyboard('{Home}{Enter}')
    expect(view.emitted()['update:modelValue']).toContainEqual([''])
  })

  it('使用外部 id 连接 label 并保留 aria-describedby', () => {
    render({
      components: { Select },
      template: `
        <label for="route-model">用途模型</label>
        <Select id="route-model" aria-describedby="route-model-help route-model-error" :options="[{ label: 'GPT', value: 'gpt' }]" />
        <p id="route-model-help">只显示可用模型</p>
        <p id="route-model-error">当前值无效</p>
      `
    })

    const trigger = screen.getByLabelText('用途模型')
    expect(trigger).toHaveAttribute('id', 'route-model')
    expect(trigger).toHaveAttribute('aria-describedby', 'route-model-help route-model-error')
  })
})
