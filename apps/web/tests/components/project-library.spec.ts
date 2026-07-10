import { render, screen } from '@testing-library/vue'
import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import ProjectLibrary from '../../features/project-library/ProjectLibrary.vue'

describe('ProjectLibrary', () => {
  it('加载失败时显示可访问错误并记录日志', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    vi.stubGlobal('watch', watch)

    render(ProjectLibrary, {
      props: {
        projects: [],
        openedProjectIds: [],
        error: '项目服务暂不可用'
      }
    })

    expect(screen.getByRole('alert')).toHaveTextContent('项目服务暂不可用')
    expect(consoleError).toHaveBeenCalled()
    consoleError.mockRestore()
  })
})
