import { describe, expect, it } from 'vitest'
import {
  isDashboardResponse,
  isErrorResponse,
  isRaceDetailResponse,
  isRaceResultsResponse,
} from './contracts'

const coverage = {
  status: 'complete',
  sourceFetchedAt: '2024-05-27T12:00:00Z',
  publishedAt: '2026-07-31T12:00:00Z',
}

const race = {
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
  coverage,
}

describe('API contract validation', () => {
  it('accepts complete and empty dashboards', () => {
    expect(isDashboardResponse({ races: [race], coverage })).toBe(true)
    expect(isDashboardResponse({ races: [], coverage: null })).toBe(true)
  })

  it('validates race sessions independently from dashboard summaries', () => {
    expect(
      isRaceDetailResponse({
        ...race,
        sessions: [
          {
            id: 'session_2024-monaco-grand-prix-race',
            name: 'Race',
            type: 'Race',
            startAt: '2024-05-26T13:00:00Z',
            endAt: '2024-05-26T15:00:00Z',
            isCancelled: false,
          },
        ],
      }),
    ).toBe(true)
  })

  it('preserves every result value kind in classification responses', () => {
    const values = [
      { kind: 'missing' },
      { kind: 'null' },
      { kind: 'number', seconds: 84.2 },
      { kind: 'text', text: '+1 Lap' },
      { kind: 'numbers', segmentsSeconds: [30.1, null, 31.2] },
    ]
    const classification = values.map((value, index) => ({
      driver: { id: `driver_${index}`, name: `Driver ${index}`, acronym: 'DRV' },
      constructor: { id: 'constructor_2024-team', name: 'Team' },
      driverNumber: index,
      position: index === 0 ? null : index,
      state: index === 0 ? 'dnf' : 'ordinary',
      laps: index === 0 ? null : 78,
      duration: value,
      gap: value,
    }))

    expect(
      isRaceResultsResponse({
        raceId: race.id,
        sessionId: 'session_2024-monaco-grand-prix-race',
        classification,
        coverage,
      }),
    ).toBe(true)
  })

  it('rejects malformed success and error payloads', () => {
    expect(
      isDashboardResponse({ races: [{ ...race, startAt: 'not-a-date' }], coverage }),
    ).toBe(false)
    expect(isErrorResponse({ error: { code: 'race_not_found' } })).toBe(false)
    expect(
      isErrorResponse({
        error: { code: 'race_not_found', message: 'Race not found.' },
      }),
    ).toBe(true)
  })
})
