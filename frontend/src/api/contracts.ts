export type Coverage = {
  status: 'complete'
  sourceFetchedAt: string
  publishedAt: string
}

export type Circuit = {
  id: string
  name: string
  countryCode: string
  countryName: string
  location: string
}

export type RaceSummary = {
  id: string
  season: number
  name: string
  officialName: string
  circuit: Circuit
  startAt: string
  endAt: string
  coverage: Coverage
}

export type DashboardResponse = {
  races: RaceSummary[]
  coverage: Coverage | null
}

export type SessionSummary = {
  id: string
  name: string
  type: string
  startAt: string
  endAt: string
  isCancelled: boolean
}

export type RaceDetailResponse = Omit<RaceSummary, 'coverage'> & {
  sessions: SessionSummary[]
  coverage: Coverage
}

export type ClassificationState =
  | 'ordinary'
  | 'dns'
  | 'dnf'
  | 'dsq'
  | 'unknown'
  | 'missing'

export type ResultValue =
  | { kind: 'missing' | 'null' }
  | { kind: 'number'; seconds: number }
  | { kind: 'text'; text: string }
  | { kind: 'numbers'; segmentsSeconds: (number | null)[] }

export type ClassificationRow = {
  driver: { id: string; name: string; acronym: string }
  constructor: { id: string; name: string }
  driverNumber: number
  position: number | null
  state: ClassificationState
  laps: number | null
  duration: ResultValue
  gap: ResultValue
}

export type RaceResultsResponse = {
  raceId: string
  sessionId: string
  classification: ClassificationRow[]
  coverage: Coverage
}

export type ErrorResponse = {
  error: {
    code: string
    message: string
  }
}

type RecordValue = Record<string, unknown>

export function isDashboardResponse(value: unknown): value is DashboardResponse {
  if (!isRecord(value) || !Array.isArray(value.races)) return false
  return (
    value.races.every(isRaceSummary) &&
    (value.coverage === null || isCoverage(value.coverage))
  )
}

export function isRaceDetailResponse(
  value: unknown,
): value is RaceDetailResponse {
  return (
    isRaceSummary(value) &&
    'sessions' in value &&
    Array.isArray(value.sessions) &&
    value.sessions.every(isSessionSummary)
  )
}

export function isRaceResultsResponse(
  value: unknown,
): value is RaceResultsResponse {
  return (
    isRecord(value) &&
    isNonEmptyString(value.raceId) &&
    isNonEmptyString(value.sessionId) &&
    Array.isArray(value.classification) &&
    value.classification.every(isClassificationRow) &&
    isCoverage(value.coverage)
  )
}

export function isErrorResponse(value: unknown): value is ErrorResponse {
  return (
    isRecord(value) &&
    isRecord(value.error) &&
    isNonEmptyString(value.error.code) &&
    isNonEmptyString(value.error.message)
  )
}

function isRaceSummary(value: unknown): value is RaceSummary {
  return (
    isRecord(value) &&
    isNonEmptyString(value.id) &&
    isInteger(value.season) &&
    isNonEmptyString(value.name) &&
    isNonEmptyString(value.officialName) &&
    isCircuit(value.circuit) &&
    isTimestamp(value.startAt) &&
    isTimestamp(value.endAt) &&
    isCoverage(value.coverage)
  )
}

function isCoverage(value: unknown): value is Coverage {
  return (
    isRecord(value) &&
    value.status === 'complete' &&
    isTimestamp(value.sourceFetchedAt) &&
    isTimestamp(value.publishedAt)
  )
}

function isCircuit(value: unknown): value is Circuit {
  return (
    isRecord(value) &&
    isNonEmptyString(value.id) &&
    isNonEmptyString(value.name) &&
    isNonEmptyString(value.countryCode) &&
    isNonEmptyString(value.countryName) &&
    isNonEmptyString(value.location)
  )
}

function isSessionSummary(value: unknown): value is SessionSummary {
  return (
    isRecord(value) &&
    isNonEmptyString(value.id) &&
    isNonEmptyString(value.name) &&
    isNonEmptyString(value.type) &&
    isTimestamp(value.startAt) &&
    isTimestamp(value.endAt) &&
    typeof value.isCancelled === 'boolean'
  )
}

function isClassificationRow(value: unknown): value is ClassificationRow {
  return (
    isRecord(value) &&
    isDriver(value.driver) &&
    isConstructor(value.constructor) &&
    isInteger(value.driverNumber) &&
    (value.position === null || isInteger(value.position)) &&
    isClassificationState(value.state) &&
    (value.laps === null || isInteger(value.laps)) &&
    isResultValue(value.duration) &&
    isResultValue(value.gap)
  )
}

function isDriver(value: unknown) {
  return (
    isRecord(value) &&
    isNonEmptyString(value.id) &&
    isNonEmptyString(value.name) &&
    isNonEmptyString(value.acronym)
  )
}

function isConstructor(value: unknown) {
  return (
    isRecord(value) &&
    isNonEmptyString(value.id) &&
    isNonEmptyString(value.name)
  )
}

function isClassificationState(value: unknown): value is ClassificationState {
  return (
    typeof value === 'string' &&
    ['ordinary', 'dns', 'dnf', 'dsq', 'unknown', 'missing'].includes(value)
  )
}

function isResultValue(value: unknown): value is ResultValue {
  if (!isRecord(value) || typeof value.kind !== 'string') return false
  switch (value.kind) {
    case 'missing':
    case 'null':
      return true
    case 'number':
      return isFiniteNumber(value.seconds)
    case 'text':
      return typeof value.text === 'string'
    case 'numbers':
      return (
        Array.isArray(value.segmentsSeconds) &&
        value.segmentsSeconds.every(
          (segment) => segment === null || isFiniteNumber(segment),
        )
      )
    default:
      return false
  }
}

function isRecord(value: unknown): value is RecordValue {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0
}

function isTimestamp(value: unknown): value is string {
  return typeof value === 'string' && !Number.isNaN(Date.parse(value))
}

function isInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value)
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}
