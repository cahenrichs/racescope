import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIError, getDashboard, getRaceDetail, getRaceResults } from './client'

const dashboard = {
  races: [],
  coverage: null,
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('race API client', () => {
  it('requests and validates the dashboard contract', async () => {
    const fetchMock = vi
      .spyOn(window, 'fetch')
      .mockResolvedValue(new Response(JSON.stringify(dashboard), { status: 200 }))

    await expect(getDashboard()).resolves.toEqual(dashboard)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/dashboard',
      expect.objectContaining({ headers: { Accept: 'application/json' } }),
    )
  })

  it('retains typed API error codes and response statuses', async () => {
    vi.spyOn(window, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'race_incomplete',
            message: 'The race classification is incomplete.',
          },
        }),
        { status: 409 },
      ),
    )

    const error = await getRaceResults('meeting_monaco').catch(
      (reason: unknown) => reason,
    )
    expect(error).toBeInstanceOf(APIError)
    expect(error).toMatchObject({
      code: 'race_incomplete',
      status: 409,
      retryable: false,
    })
  })

  it('times out stalled requests with an actionable error', async () => {
    vi.useFakeTimers()
    vi.spyOn(window, 'fetch').mockImplementation(
      (_input, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'))
          })
        }),
    )

    const request = getDashboard({ timeoutMs: 50 })
    const expectation = expect(request).rejects.toMatchObject({
      code: 'request_timeout',
      retryable: true,
    })
    await vi.advanceTimersByTimeAsync(50)
    await expectation
  })

  it('cancels requests and encodes race IDs in both race URLs', async () => {
    const controller = new AbortController()
    const fetchMock = vi.spyOn(window, 'fetch').mockImplementation(
      (_input, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('Aborted', 'AbortError'))
          })
        }),
    )

    const detail = getRaceDetail('meeting/monaco', {
      signal: controller.signal,
    })
    const results = getRaceResults('meeting/monaco', {
      signal: controller.signal,
    })
    controller.abort()

    await expect(detail).rejects.toMatchObject({ code: 'request_cancelled' })
    await expect(results).rejects.toMatchObject({ code: 'request_cancelled' })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/races/meeting%2Fmonaco')
    expect(fetchMock.mock.calls[1]?.[0]).toBe(
      '/api/races/meeting%2Fmonaco/results',
    )
  })
})
