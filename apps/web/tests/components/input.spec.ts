import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Input from '../../components/ui/Input.vue'

describe('UiInput', () => {
  it('运行时保持直角并转发输入值', async () => {
    const user = userEvent.setup()
    const view = render(Input, {
      props: { modelValue: '初稿' },
      attrs: { 'aria-label': '章节标题' }
    })

    const input = screen.getByRole('textbox', { name: '章节标题' })
    expect(getComputedStyle(input).borderRadius).toBe('0px')
    await user.clear(input)
    await user.type(input, '定稿')
    expect(view.emitted()['update:modelValue']?.at(-1)).toEqual(['定稿'])
  })
})
