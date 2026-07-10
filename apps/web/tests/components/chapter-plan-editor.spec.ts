import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import ChapterPlanEditor from '../../features/chapter-plan/ChapterPlanEditor.vue'
import Button from '../../components/ui/Button.vue'
import Input from '../../components/ui/Input.vue'
import Select from '../../components/ui/Select.vue'
import Textarea from '../../components/ui/Textarea.vue'

describe('ChapterPlanEditor', () => {
  it('新增规划只发出 chapter_plan 更新', async () => {
    const user = userEvent.setup()
    const view = render(ChapterPlanEditor, {
      props: { modelValue: [] },
      global: {
        components: { UiButton: Button, UiInput: Input, UiSelect: Select, UiTextarea: Textarea }
      }
    })

    await user.click(screen.getByRole('button', { name: '新增规划' }))

    const updates = view.emitted('update:modelValue') as unknown[][]
    expect(updates).toHaveLength(1)
    expect(updates[0]?.[0]).toEqual([
      { id: 'chapter-plan-1', title: '', status: 'planned', summary: '' }
    ])
  })
})
