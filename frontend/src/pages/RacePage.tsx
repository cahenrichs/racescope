import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { APIError, getRaceDetail, getRaceResults } from '../api/client'
import type {
  ClassificationState,
  RaceDetailResponse,
  RaceResultsResponse,
  ResultValue,
} from '../api/contracts'

type RacePageProps = {
  loadDetail?: typeof getRaceDetail
  loadResults?: typeof getRaceResults
}

type ResourceState<T> =
  | { kind: 'loading' }
  | { kind: 'loaded'; value: T }
  | { kind: 'error'; error: APIError | Error }

const dateFormatter = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})

const dateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'UTC',
})

const stateLabels: Record<ClassificationState, string> = {
  ordinary: 'Classified',
  dns: 'DNS',
  dnf: 'DNF',
  dsq: 'DSQ',
  unknown: 'Unknown',
  missing: 'Not available',
}

export default function RacePage({
  loadDetail = getRaceDetail,
  loadResults = getRaceResults,
}: RacePageProps) {
  const { meetingID } = useParams()
  const [detailAttempt, setDetailAttempt] = useState(0)
  const [resultsAttempt, setResultsAttempt] = useState(0)
  const [detail, setDetail] = useState<ResourceState<RaceDetailResponse>>({
    kind: 'loading',
  })
  const [results, setResults] = useState<ResourceState<RaceResultsResponse>>({
    kind: 'loading',
  })

  useEffect(() => {
    if (!meetingID) return

    const controller = new AbortController()
    let active = true
    loadDetail(meetingID, { signal: controller.signal }).then(
      (value) => active && setDetail({ kind: 'loaded', value }),
      (error: unknown) =>
        active && setDetail({ kind: 'error', error: toError(error) }),
    )

    return () => {
      active = false
      controller.abort()
    }
  }, [detailAttempt, loadDetail, meetingID])

  useEffect(() => {
    if (!meetingID) return

    const controller = new AbortController()
    let active = true
    loadResults(meetingID, { signal: controller.signal }).then(
      (value) => active && setResults({ kind: 'loaded', value }),
      (error: unknown) =>
        active && setResults({ kind: 'error', error: toError(error) }),
    )

    return () => {
      active = false
      controller.abort()
    }
  }, [loadResults, meetingID, resultsAttempt])

  function retryDetail() {
    setDetail({ kind: 'loading' })
    setDetailAttempt((attempt) => attempt + 1)
  }

  function retryResults() {
    setResults({ kind: 'loading' })
    setResultsAttempt((attempt) => attempt + 1)
  }

  if (!meetingID) {
    return <PageError title="Race not found" message="No race was specified." />
  }

  if (detail.kind === 'loading') {
    return (
      <main>
        <BackLink />
        <div className="race-status" role="status" aria-live="polite">
          <p className="eyebrow">Loading weekend</p>
          <h1>Retrieving race details.</h1>
        </div>
      </main>
    )
  }

  if (detail.kind === 'error') {
    const notFound =
      detail.error instanceof APIError && detail.error.code === 'race_not_found'
    return (
      <PageError
        title={notFound ? 'Race not found' : 'Race details unavailable'}
        message={detail.error.message}
        retry={notFound ? undefined : retryDetail}
      />
    )
  }

  const race = detail.value

  return (
    <main>
      <BackLink />
      <header className="race-hero">
        <div>
          <p className="eyebrow">
            {race.season} / {race.circuit.countryName}
          </p>
          <h1>{race.name}</h1>
          <p className="race-hero__official">{race.officialName}</p>
        </div>
        <dl className="race-facts">
          <div>
            <dt>Circuit</dt>
            <dd>{race.circuit.name}</dd>
          </div>
          <div>
            <dt>Location</dt>
            <dd>
              {race.circuit.location}, {race.circuit.countryName}
            </dd>
          </div>
          <div>
            <dt>Weekend</dt>
            <dd>
              <time dateTime={race.startAt}>
                {dateFormatter.format(new Date(race.startAt))}
              </time>{' '}
              to{' '}
              <time dateTime={race.endAt}>
                {dateFormatter.format(new Date(race.endAt))}
              </time>
            </dd>
          </div>
          <div>
            <dt>Published</dt>
            <dd>
              <time dateTime={race.coverage.publishedAt}>
                {dateTimeFormatter.format(new Date(race.coverage.publishedAt))}
              </time>
            </dd>
          </div>
        </dl>
      </header>

      <section className="race-section" aria-labelledby="sessions-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Full schedule</p>
            <h2 id="sessions-heading">Weekend sessions</h2>
          </div>
          <p>{race.sessions.length} sessions</p>
        </div>
        <ol className="session-list">
          {race.sessions.map((session) => (
            <li key={session.id}>
              <div>
                <h3>{session.name}</h3>
                <p>{session.type}</p>
              </div>
              <p>
                <time dateTime={session.startAt}>
                  {dateTimeFormatter.format(new Date(session.startAt))}
                </time>
                {' - '}
                <time dateTime={session.endAt}>
                  {dateTimeFormatter.format(new Date(session.endAt))}
                </time>
              </p>
              {session.isCancelled && <strong>Cancelled</strong>}
            </li>
          ))}
        </ol>
      </section>

      <section className="race-section" aria-labelledby="classification-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Grand Prix result</p>
            <h2 id="classification-heading">Classification</h2>
          </div>
        </div>
        <ClassificationRegion state={results} retry={retryResults} />
      </section>
    </main>
  )
}

function ClassificationRegion({
  state,
  retry,
}: {
  state: ResourceState<RaceResultsResponse>
  retry: () => void
}) {
  if (state.kind === 'loading') {
    return (
      <div className="inline-status" role="status" aria-live="polite" aria-busy="true">
        Loading race classification.
      </div>
    )
  }

  if (state.kind === 'error') {
    const incomplete =
      state.error instanceof APIError && state.error.code === 'race_incomplete'
    return (
      <div className="inline-status inline-status--error" role="alert">
        <h3>{incomplete ? 'Classification incomplete' : 'Classification unavailable'}</h3>
        <p>{state.error.message}</p>
        <button type="button" onClick={retry}>
          Retry classification
        </button>
      </div>
    )
  }

  return (
    <div className="classification-scroll">
      <table className="classification-table">
        <caption>Grand Prix final classification</caption>
        <thead>
          <tr>
            <th scope="col">Pos.</th>
            <th scope="col">Driver</th>
            <th scope="col">Constructor</th>
            <th scope="col">Laps</th>
            <th scope="col">Time</th>
            <th scope="col">Gap</th>
          </tr>
        </thead>
        <tbody>
          {state.value.classification.map((row) => (
            <tr key={row.driver.id}>
              <th scope="row">
                {row.position ?? stateLabels[row.state]}
                {row.position !== null && row.state !== 'ordinary' && (
                  <span className="classification-state"> {stateLabels[row.state]}</span>
                )}
              </th>
              <td>
                <span className="driver-number">{row.driverNumber}</span>
                <strong>{row.driver.name}</strong>
                <span className="driver-acronym">{row.driver.acronym}</span>
              </td>
              <td>{row.constructor.name}</td>
              <td>{row.laps ?? 'Not available'}</td>
              <td>{formatResultValue(row.duration)}</td>
              <td>{formatResultValue(row.gap)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function PageError({
  title,
  message,
  retry,
}: {
  title: string
  message: string
  retry?: () => void
}) {
  return (
    <main>
      <BackLink />
      <div className="race-status race-status--error" role="alert">
        <p className="eyebrow">Unable to show weekend</p>
        <h1>{title}</h1>
        <p>{message}</p>
        {retry && (
          <button type="button" onClick={retry}>
            Retry race details
          </button>
        )}
      </div>
    </main>
  )
}

function BackLink() {
  return (
    <Link className="back-link" to="/">
      Back to race archive
    </Link>
  )
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error('An unexpected error occurred.')
}

function formatResultValue(value: ResultValue) {
  switch (value.kind) {
    case 'missing':
      return 'Not recorded'
    case 'null':
      return 'Not available'
    case 'text':
      return value.text
    case 'number':
      return formatSeconds(value.seconds)
    case 'numbers':
      return value.segmentsSeconds
        .map((segment) => (segment === null ? 'Not available' : formatSeconds(segment)))
        .join(' / ')
  }
}

function formatSeconds(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = (totalSeconds % 60).toFixed(3).padStart(6, '0')

  if (hours > 0) return `${hours}:${String(minutes).padStart(2, '0')}:${seconds}`
  if (minutes > 0) return `${minutes}:${seconds}`
  return `${seconds}s`
}
