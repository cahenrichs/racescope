const readinessTimeoutMs = 5_000

export type ReadinessResponse = {
  status: 'ok'
}

type ReadinessOptions = {
  signal?: AbortSignal
  timeoutMs?: number
}

export async function getReadiness({
  signal,
  timeoutMs = readinessTimeoutMs,
}: ReadinessOptions = {}): Promise<ReadinessResponse> {
  const controller = new AbortController()
  let timedOut = false
  const cancelRequest = () => controller.abort()
  const timeout = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, timeoutMs)

  signal?.addEventListener('abort', cancelRequest, { once: true })

  let response: Response
  try {
    response = await fetch('/ready', {
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })
  } catch (error) {
    if (timedOut) {
      throw new Error('The readiness check timed out.', { cause: error })
    }
    if (signal?.aborted) {
      throw new Error('The readiness check was cancelled.', { cause: error })
    }
    throw new Error('The API could not be reached.', { cause: error })
  } finally {
    window.clearTimeout(timeout)
    signal?.removeEventListener('abort', cancelRequest)
  }

  if (!response.ok) {
    throw new Error(`The API reported that it is unavailable (${response.status}).`)
  }

  let payload: unknown
  try {
    payload = await response.json()
  } catch (error) {
    throw new Error('The API returned an invalid readiness response.', {
      cause: error,
    })
  }

  if (!isReadinessResponse(payload)) {
    throw new Error('The API returned an invalid readiness response.')
  }

  return payload
}

function isReadinessResponse(value: unknown): value is ReadinessResponse {
  return (
    typeof value === 'object' &&
    value !== null &&
    'status' in value &&
    value.status === 'ok'
  )
}
