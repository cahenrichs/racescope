import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from './App'
import type { ReadinessResponse } from './api/client'

describe('App', () => {
  it('renders an accessible loading state while readiness is pending', () => {
    const loadReadiness = vi.fn(
      () => new Promise<ReadinessResponse>(() => undefined),
    )

    render(<App loadReadiness={loadReadiness} />)

    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByText('Checking the pit wall.')).toBeInTheDocument()
  })

  it('renders the ready state when the API and database are available', async () => {
    const loadReadiness = vi.fn(async () => ({ status: 'ok' as const }))

    render(<App loadReadiness={loadReadiness} />)

    expect(await screen.findByText('Systems are ready.')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'false')
    expect(
      screen.getByText('The API is running and connected to PostgreSQL.'),
    ).toBeInTheDocument()
  })

  it('renders an actionable unavailable state when readiness fails', async () => {
    const loadReadiness = vi.fn(async () => {
      throw new Error('The API could not be reached.')
    })

    render(<App loadReadiness={loadReadiness} />)

    expect(
      await screen.findByText('RaceScope is unavailable.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(
      'The API could not be reached.',
    )
    expect(screen.getByRole('button', { name: 'Try again' })).toBeEnabled()
  })
})
