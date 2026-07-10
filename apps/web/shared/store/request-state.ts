import type { ApiErrorState, ApiRequestState } from '~/shared/api'
import { toApiErrorState } from '~/shared/api'

export function createApiRequestState(): ApiRequestState {
  return { loading: false, error: null }
}

export async function withApiRequestState<T>(
  state: ApiRequestState,
  endpoint: string,
  operation: () => Promise<T>
): Promise<T> {
  state.loading = true
  state.error = null
  try {
    return await operation()
  } catch (cause) {
    const error: ApiErrorState = toApiErrorState(endpoint, cause)
    console.error(`[AeonEchoes Store] ${endpoint} failed`, error)
    state.error = error
    throw cause
  } finally {
    state.loading = false
  }
}
