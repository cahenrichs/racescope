import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import App from './App'
import type { DashboardResponse, RaceSummary } from './api/contracts'

const monaco: RaceSummary = {
  id: 'meeting_2024-monaco-grand-prix',
  season: 2024,
  name: 'Monaco Grand Prix',
  officialName: 'FORMULA 1 GRAND PRIX DE MONACO 2024',
  circuit: {
    id: 'circuit_circuit-de-monaco',
    name: 'Circuit de Monaco',
    countryCode: 'MON',
    countryName: 'Monaco',
    location: 'Monte Carlo',
  },
  startAt: '2024-05-24T11:30:00Z',
  endAt: '2024-05-26T15:00:00Z',
  coverage: {
    status: 'complete',
    sourceFetchedAt: '2024-05-27T12:00:00Z',
    publishedAt: '2026-07-31T12:00:00Z',
  },
}

const dashboard: DashboardResponse = {
  races: [
    monaco,
    {
      ...monaco,
      id: 'meeting_2024-british-grand-prix',
      name: 'British Grand Prix',
      officialName: 'FORMULA 1 BRITISH GRAND PRIX 2024',
    },
  ],
  coverage: monaco.coverage,
}

describe('App', () => {
  it('renders an accessible loading state while the dashboard is pending', () => {
    const loadDashboard = vi.fn(
      () => new Promise<DashboardResponse>(() => undefined),
    )

    renderApp(loadDashboard)

    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByText('Checking the timing screens.')).toBeInTheDocument()
  })

  it('renders every race with coverage dates and deterministic links', async () => {
    const loadDashboard = vi.fn(async () => dashboard)

    renderApp(loadDashboard)

    const monacoLink = await screen.findByRole('link', {
      name: 'Monaco Grand Prix',
    })
    expect(monacoLink).toHaveAttribute(
      'href',
      '/races/meeting_2024-monaco-grand-prix',
    )
    expect(
      screen.getByRole('link', { name: 'British Grand Prix' }),
    ).toHaveAttribute('href', '/races/meeting_2024-british-grand-prix')
    expect(screen.getAllByText('Complete')).toHaveLength(2)
    expect(screen.getAllByText(/Source updated/)).toHaveLength(2)
    expect(screen.getAllByText(/^Published /)).toHaveLength(2)
  })

  it('renders the empty dashboard state', async () => {
    const loadDashboard = vi.fn(async () => ({ races: [], coverage: null }))

    renderApp(loadDashboard)

    expect(
      await screen.findByText('No race weekends are published yet.'),
    ).toBeInTheDocument()
  })

  it('renders an actionable error when the dashboard fails', async () => {
    const loadDashboard = vi.fn(async () => {
      throw new Error('The dashboard could not be reached.')
    })

    renderApp(loadDashboard)

    expect(
      await screen.findByText('The race archive is unavailable.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(
      'The dashboard could not be reached.',
    )
    expect(screen.getByRole('button', { name: 'Try again' })).toBeEnabled()
  })
})

function renderApp(loadDashboard: NonNullable<Parameters<typeof App>[0]['loadDashboard']>) {
  return render(
    <MemoryRouter>
      <App loadDashboard={loadDashboard} />
    </MemoryRouter>,
  )
}
