import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import Field from '../../components/ui/Field.vue'

const FieldHarness = {
  components: { Field },
  template: `
    <Field label="名称" description="用于展示" error="名称不能为空" required>
      <template #default="slotProps">
        <input v-bind="{ id: slotProps.id, 'aria-describedby': slotProps.describedby, 'aria-invalid': slotProps.invalid }">
      </template>
    </Field>
  `
}

describe('UiField', () => {
  it('关联标签、描述和错误消息', () => {
    render(FieldHarness)
    const input = screen.getByRole('textbox', { name: /名称/ })
    expect(input).toHaveAttribute('aria-invalid', 'true')
    const describedBy = input.getAttribute('aria-describedby') || ''
    expect(describedBy.split(' ')).toHaveLength(2)
    expect(screen.getByRole('alert')).toHaveTextContent('名称不能为空')
  })
})
