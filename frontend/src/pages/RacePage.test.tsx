import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { APIError } from '../api/client'
import type { RaceDetailResponse, RaceResultsResponse } from '../api/contracts'
import { raceDetail, raceResults } from '../test/fixtures'
import RacePage from './RacePage'

describe('RacePage', () => {
  it('announces loading while race detail is pending', () => {
    renderRacePage({
      loadDetail: vi.fn(() => pending<RaceDetailResponse>()),
      loadResults: vi.fn(() => pending<RaceResultsResponse>()),
    })

    expect(screen.getByRole('status')).toHaveTextContent('Retrieving race details.')
  })

  it('renders race metadata, sessions, and the semantic classification table', async () => {
    renderRacePage()

    expect(
      await screen.findByRole('heading', { name: 'Monaco Grand Prix', level: 1 }),
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Weekend sessions' })).toBeInTheDocument()
    expect(screen.getByText('Practice 1')).toBeInTheDocument()
    const table = await screen.findByRole('table', {
      name: 'Grand Prix final classification',
    })
    expect(table).toHaveTextContent('Charles Leclerc')
    expect(table).toHaveTextContent('Max Verstappen')
    expect(screen.getByRole('columnheader', { name: 'Driver' })).toBeInTheDocument()
  })

  it('renders race not found without an inappropriate retry action', async () => {
    const loadDetail = vi.fn(async () => {
      throw new APIError('That race has not been published.', {
        status: 404,
        code: 'race_not_found',
        retryable: false,
      })
    })

    renderRacePage({ loadDetail })

    expect(
      await screen.findByRole('heading', { name: 'Race not found' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /retry race details/i })).not.toBeInTheDocument()
  })

  it('keeps valid detail visible and retries an incomplete classification', async () => {
    const user = userEvent.setup()
    const incomplete = new APIError('The race classification is incomplete.', {
      status: 409,
      code: 'race_incomplete',
      retryable: false,
    })
    const loadResults = vi
      .fn<NonNullable<Parameters<typeof RacePage>[0]['loadResults']>>()
      .mockRejectedValueOnce(incomplete)
      .mockResolvedValueOnce(raceResults)

    renderRacePage({ loadResults })

    expect(
      await screen.findByRole('heading', { name: 'Classification incomplete' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Monaco Grand Prix' })).toBeInTheDocument()
    expect(screen.getByText('Practice 1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Retry classification' }))

    expect(
      await screen.findByRole('table', { name: 'Grand Prix final classification' }),
    ).toHaveTextContent('Charles Leclerc')
    expect(loadResults).toHaveBeenCalledTimes(2)
  })

  it('shows a page-level retry for a race detail server error', async () => {
    const user = userEvent.setup()
    const loadDetail = vi
      .fn<NonNullable<Parameters<typeof RacePage>[0]['loadDetail']>>()
      .mockRejectedValueOnce(
        new APIError('The race detail service is unavailable.', {
          status: 503,
          code: 'server_error',
          retryable: true,
        }),
      )
      .mockResolvedValueOnce(raceDetail)

    renderRacePage({ loadDetail })

    await user.click(await screen.findByRole('button', { name: 'Retry race details' }))

    expect(
      await screen.findByRole('heading', { name: 'Monaco Grand Prix' }),
    ).toBeInTheDocument()
    expect(loadDetail).toHaveBeenCalledTimes(2)
  })
})

function renderRacePage({
  loadDetail = vi.fn(async () => raceDetail),
  loadResults = vi.fn(async () => raceResults),
}: {
  loadDetail?: NonNullable<Parameters<typeof RacePage>[0]['loadDetail']>
  loadResults?: NonNullable<Parameters<typeof RacePage>[0]['loadResults']>
} = {}) {
  return render(
    <MemoryRouter initialEntries={[`/races/${raceDetail.id}`]}>
      <Routes>
        <Route
          path="/races/:meetingID"
          element={<RacePage loadDetail={loadDetail} loadResults={loadResults} />}
        />
      </Routes>
    </MemoryRouter>,
  )
}

function pending<T>() {
  return new Promise<T>(() => undefined)
}
