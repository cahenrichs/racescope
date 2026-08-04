import type {
  DashboardResponse,
  RaceDetailResponse,
  RaceResultsResponse,
  RaceSummary,
} from '../api/contracts'

export const monaco: RaceSummary = {
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

export const dashboard: DashboardResponse = {
  races: [monaco],
  coverage: monaco.coverage,
}

export const raceDetail: RaceDetailResponse = {
  ...monaco,
  sessions: [
    {
      id: 'session_2024-monaco-grand-prix-practice-1',
      name: 'Practice 1',
      type: 'Practice',
      startAt: '2024-05-24T11:30:00Z',
      endAt: '2024-05-24T12:30:00Z',
      isCancelled: false,
    },
    {
      id: 'session_2024-monaco-grand-prix-race',
      name: 'Race',
      type: 'Race',
      startAt: '2024-05-26T13:00:00Z',
      endAt: '2024-05-26T15:00:00Z',
      isCancelled: false,
    },
  ],
}

export const raceResults: RaceResultsResponse = {
  raceId: monaco.id,
  sessionId: 'session_2024-monaco-grand-prix-race',
  classification: [
    {
      driver: {
        id: 'driver_charles-leclerc',
        name: 'Charles Leclerc',
        acronym: 'LEC',
      },
      constructor: {
        id: 'constructor_2024-ferrari',
        name: 'Ferrari',
      },
      driverNumber: 16,
      position: 1,
      state: 'ordinary',
      laps: 78,
      duration: { kind: 'number', seconds: 8342.820 },
      gap: { kind: 'number', seconds: 0 },
    },
    {
      driver: {
        id: 'driver_max-verstappen',
        name: 'Max Verstappen',
        acronym: 'VER',
      },
      constructor: {
        id: 'constructor_2024-red-bull-racing',
        name: 'Red Bull Racing',
      },
      driverNumber: 1,
      position: null,
      state: 'dnf',
      laps: 2,
      duration: { kind: 'null' },
      gap: { kind: 'text', text: 'DNF' },
    },
  ],
  coverage: monaco.coverage,
}
