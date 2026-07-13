import { render, screen, within } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import ChapterCreateDialog from '../../features/chapter-create/ChapterCreateDialog.vue'
import Button from '../../components/ui/Button.vue'
import Dialog from '../../components/ui/Dialog.vue'
import Input from '../../components/ui/Input.vue'
import Select from '../../components/ui/Select.vue'
import Textarea from '../../components/ui/Textarea.vue'

const global = {
  components: {
    UiButton: Button,
    UiDialog: Dialog,
    UiInput: Input,
    UiSelect: Select,
    UiTextarea: Textarea
  },
  provide: {}
}

describe('ChapterCreateDialog', () => {
  it('章节状态选项完整使用统一中文键', async () => {
    const user = userEvent.setup()
    render(ChapterCreateDialog, {
      props: { open: true, chapters: [] },
      global: { ...global, stubs: { Teleport: true } }
    })

    await user.click(screen.getByRole('button', { name: '状态' }))
    const options = within(screen.getByRole('listbox', { name: '状态' }))
    expect(options.getByText('计划中')).toBeVisible()
    expect(options.getByText('写作中')).toBeVisible()
    expect(options.getByText('审阅中')).toBeVisible()
    expect(options.getByText('已锁定')).toBeVisible()
  })

  it('取消不会发出确认事件', async () => {
    const user = userEvent.setup()
    const view = render(ChapterCreateDialog, {
      props: { open: true, chapters: [] },
      global: { ...global, stubs: { Teleport: true } }
    })

    await user.click(screen.getByRole('button', { name: '取消' }))

    expect(view.emitted().confirm).toBeUndefined()
    expect(view.emitted()['update:open']).toContainEqual([false])
  })

  it('填写并确认后才发出真实章节创建请求', async () => {
    const user = userEvent.setup()
    const view = render(ChapterCreateDialog, {
      props: { open: true, chapters: [] },
      global: { ...global, stubs: { Teleport: true } }
    })

    expect(view.emitted().confirm).toBeUndefined()
    await user.type(screen.getByRole('textbox', { name: '章节标题' }), '第一章')
    await user.click(screen.getByRole('button', { name: '确认新建章节' }))

    expect(view.emitted().confirm).toEqual([[
      { number: 1, title: '第一章', status: 'drafting', summary: undefined }
    ]])
    expect(screen.queryByText('引用章节规划')).not.toBeInTheDocument()
  })
})
