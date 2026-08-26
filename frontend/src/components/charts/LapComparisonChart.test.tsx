import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, test } from 'vitest'
import type { LapComparisonResult } from '../../api/contracts'
import LapComparisonChart from './LapComparisonChart'

const result: LapComparisonResult = {
  sessionId: 'session_2024-monaco-grand-prix-race',
  driverIds: ['driver_charles-leclerc', 'driver_oscar-piastri'],
  title: 'Charles Leclerc and Oscar Piastri lap comparison',
  dimension: 'lap',
  units: 'microseconds',
  preferredChartType: 'line',
  series: [
    makeSeries('driver_charles-leclerc', 'Charles Leclerc', '#E10600', 'solid', 'circle', [
      74_123_456,
      null,
      73_900_000,
    ]),
    makeSeries('driver_oscar-piastri', 'Oscar Piastri', '#3671C6', 'dashed', 'square', [
      74_500_000,
      74_200_000,
      74_000_000,
    ]),
  ],
}

test('renders the production lap-series shape with redundant chart styles', () => {
  const { container } = render(<LapComparisonChart result={result} />)

  expect(screen.getByRole('heading', { name: result.title })).toBeInTheDocument()
  expect(screen.getByText(/Charles Leclerc, solid line with circle markers/)).toBeInTheDocument()
  expect(screen.getByText(/Oscar Piastri, dashed line with square markers/)).toBeInTheDocument()
  expect(container.querySelectorAll('.recharts-line-curve')).toHaveLength(2)
  expect(container.querySelector('[stroke-dasharray="8 6"]')).toBeInTheDocument()
})

test('uses one chart focus target and exposes tooltips with arrow keys', async () => {
  const { container } = render(<LapComparisonChart result={result} />)
  const chart = screen.getByRole('application', { name: result.title })

  expect(chart).toHaveAttribute('tabindex', '0')
  expect(chart).toHaveAttribute('aria-keyshortcuts', 'ArrowLeft ArrowRight Enter')
  expect(container.querySelectorAll('[tabindex="0"]')).toHaveLength(1)

  chart.focus()
  fireEvent.keyDown(chart, { key: 'ArrowRight' })

  await waitFor(() => {
    expect(screen.getByText('Lap 2')).toBeInTheDocument()
    expect(screen.getByText('1:14.200')).toBeInTheDocument()
  })

  fireEvent.keyDown(chart, { key: 'ArrowLeft' })

  await waitFor(() => {
    expect(screen.getByText('Lap 1')).toBeInTheDocument()
    expect(screen.getByText('1:14.123')).toBeInTheDocument()
    expect(screen.getByText('1:14.500')).toBeInTheDocument()
  })
})

function makeSeries(
  id: string,
  name: string,
  colour: string,
  lineStyle: 'solid' | 'dashed',
  marker: 'circle' | 'square',
  durations: (number | null)[],
) {
  return {
    driver: { id, name, acronym: name.split(' ').map((part) => part[0]).join('') },
    style: { colour, lineStyle, marker },
    coverage: { seriesId: id, status: 'complete' as const, fields: [] },
    observations: durations.map((durationMicroseconds, index) => ({
      lapNumber: index + 1,
      durationMicroseconds,
      missingReason:
        durationMicroseconds === null ? ('source-duration-missing' as const) : null,
      compound: 'HARD',
      stintNumber: 1,
      isPitOutLap: false,
      isStintStart: index === 0,
      isStintEnd: index === durations.length - 1,
    })),
  }
}
