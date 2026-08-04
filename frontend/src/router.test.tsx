import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { routes } from './router'
import { dashboard, raceDetail, raceResults } from './test/fixtures'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('dashboard-to-result routing', () => {
  it('navigates through the real routes and supports keyboard navigation back', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.spyOn(window, 'fetch').mockImplementation(async (input) => {
      const path = String(input)
      const payload =
        path === '/api/dashboard'
          ? dashboard
          : path.endsWith('/results')
            ? raceResults
            : raceDetail
      return new Response(JSON.stringify(payload), { status: 200 })
    })
    const router = createMemoryRouter(routes, { initialEntries: ['/'] })

    render(<RouterProvider router={router} />)

    await user.click(await screen.findByRole('link', { name: 'Monaco Grand Prix' }))

    expect(
      await screen.findByRole('table', { name: 'Grand Prix final classification' }),
    ).toHaveTextContent('Charles Leclerc')
    expect(router.state.location.pathname).toBe(`/races/${raceDetail.id}`)
    expect(fetchMock).toHaveBeenCalledTimes(3)

    await user.tab()
    const backLink = screen.getByRole('link', { name: 'Back to race archive' })
    expect(backLink).toHaveFocus()
    await user.keyboard('{Enter}')

    expect(await screen.findByRole('link', { name: 'Monaco Grand Prix' })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/')
  })
})
