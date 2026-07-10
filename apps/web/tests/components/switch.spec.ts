import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import Switch from '../../components/ui/Switch.vue'

describe('UiSwitch', () => {
  it('运行时保持开关、轨道与滑块直角并可切换', async () => {
    const user = userEvent.setup()
    const view = render(Switch, { props: { label: '启用流式输出', modelValue: false } })

    const control = screen.getByRole('switch', { name: '启用流式输出' })
    const squareParts = control.querySelectorAll<HTMLElement>('[data-aeon-square]')
    expect(getComputedStyle(control).borderRadius).toBe('0px')
    expect(squareParts).toHaveLength(2)
    for (const part of squareParts) expect(getComputedStyle(part).borderRadius).toBe('0px')

    await user.click(control)
    expect(view.emitted()['update:modelValue']).toContainEqual([true])
  })
})
