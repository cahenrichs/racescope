import {
  isDashboardResponse,
  isErrorResponse,
  isRaceDetailResponse,
  isRaceResultsResponse,
  type DashboardResponse,
  type RaceDetailResponse,
  type RaceResultsResponse,
} from './contracts'

const readinessTimeoutMs = 5_000
const apiTimeoutMs = 10_000

export type ReadinessResponse = {
  status: 'ok'
}

type ReadinessOptions = {
  signal?: AbortSignal
  timeoutMs?: number
}

export type RequestOptions = ReadinessOptions

export class APIError extends Error {
  readonly status: number | null
  readonly code: string
  readonly retryable: boolean

  constructor(
    message: string,
    options: {
      status?: number | null
      code: string
      retryable: boolean
      cause?: unknown
    },
  ) {
    super(message, { cause: options.cause })
    this.name = 'APIError'
    this.status = options.status ?? null
    this.code = options.code
    this.retryable = options.retryable
  }
}

export function getDashboard(
  options?: RequestOptions,
): Promise<DashboardResponse> {
  return getAPIResource(
    '/api/dashboard',
    'dashboard',
    isDashboardResponse,
    options,
  )
}

export function getRaceDetail(
  meetingID: string,
  options?: RequestOptions,
): Promise<RaceDetailResponse> {
  return getAPIResource(
    `/api/races/${encodeURIComponent(meetingID)}`,
    'race detail',
    isRaceDetailResponse,
    options,
  )
}

export function getRaceResults(
  meetingID: string,
  options?: RequestOptions,
): Promise<RaceResultsResponse> {
  return getAPIResource(
    `/api/races/${encodeURIComponent(meetingID)}/results`,
    'race classification',
    isRaceResultsResponse,
    options,
  )
}

async function getAPIResource<T>(
  path: string,
  resourceName: string,
  validate: (value: unknown) => value is T,
  { signal, timeoutMs = apiTimeoutMs }: RequestOptions = {},
): Promise<T> {
  const controller = new AbortController()
  let timedOut = false
  const cancelRequest = () => controller.abort()
  const timeout = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, timeoutMs)

  signal?.addEventListener('abort', cancelRequest, { once: true })
  if (signal?.aborted) controller.abort()

  let response: Response
  try {
    response = await fetch(path, {
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })
  } catch (error) {
    if (timedOut) {
      throw new APIError(`The ${resourceName} request timed out.`, {
        code: 'request_timeout',
        retryable: true,
        cause: error,
      })
    }
    if (signal?.aborted) {
      throw new APIError(`The ${resourceName} request was cancelled.`, {
        code: 'request_cancelled',
        retryable: false,
        cause: error,
      })
    }
    throw new APIError(`The ${resourceName} could not be reached.`, {
      code: 'network_error',
      retryable: true,
      cause: error,
    })
  } finally {
    window.clearTimeout(timeout)
    signal?.removeEventListener('abort', cancelRequest)
  }

  let payload: unknown
  try {
    payload = await response.json()
  } catch (error) {
    throw new APIError(`The API returned invalid JSON for the ${resourceName}.`, {
      status: response.status,
      code: 'invalid_response',
      retryable: isRetryableStatus(response.status),
      cause: error,
    })
  }

  if (!response.ok) {
    if (isErrorResponse(payload)) {
      throw new APIError(payload.error.message, {
        status: response.status,
        code: payload.error.code,
        retryable: isRetryableStatus(response.status),
      })
    }
    throw new APIError(
      `The API could not load the ${resourceName} (${response.status}).`,
      {
        status: response.status,
        code: 'http_error',
        retryable: isRetryableStatus(response.status),
      },
    )
  }

  if (!validate(payload)) {
    throw new APIError(`The API returned an invalid ${resourceName} response.`, {
      status: response.status,
      code: 'invalid_response',
      retryable: false,
    })
  }

  return payload
}

function isRetryableStatus(status: number) {
  return status === 408 || status === 429 || status >= 500
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
