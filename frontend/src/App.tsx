import { useEffect, useState } from 'react'
import { getDashboard } from './api/client'
import type { DashboardResponse } from './api/contracts'

type AppProps = {
  loadDashboard?: typeof getDashboard
}

type DashboardState =
  | { kind: 'loading' }
  | { kind: 'loaded'; dashboard: DashboardResponse }
  | { kind: 'error'; message: string }

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

export default function App({ loadDashboard = getDashboard }: AppProps) {
  const [attempt, setAttempt] = useState(0)
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    kind: 'loading',
  })

  useEffect(() => {
    const controller = new AbortController()
    let active = true

    loadDashboard({ signal: controller.signal }).then(
      (dashboard) => {
        if (active) setDashboardState({ kind: 'loaded', dashboard })
      },
      (error: unknown) => {
        if (active) {
          setDashboardState({
            kind: 'error',
            message:
              error instanceof Error
                ? error.message
                : 'The dashboard could not be loaded.',
          })
        }
      },
    )

    return () => {
      active = false
      controller.abort()
    }
  }, [attempt, loadDashboard])

  function retry() {
    setDashboardState({ kind: 'loading' })
    setAttempt((currentAttempt) => currentAttempt + 1)
  }

  return (
    <main>
      <header className="hero">
        <p className="eyebrow">RaceScope / Grand Prix archive</p>
        <h1>Race weekends, fully classified.</h1>
        <p className="intro">
          Published Formula 1 weekends with complete session coverage and race
          results.
        </p>
      </header>

      <section
        className={`dashboard dashboard--${dashboardState.kind}`}
        aria-live="polite"
        aria-busy={dashboardState.kind === 'loading'}
      >
        {dashboardState.kind === 'loading' && (
          <div className="dashboard-message" role="status" aria-busy="true">
            <p className="eyebrow">Loading archive</p>
            <h2>Checking the timing screens.</h2>
            <p>Loading published race weekends.</p>
          </div>
        )}

        {dashboardState.kind === 'error' && (
          <div className="dashboard-message dashboard-message--error" role="alert">
            <p className="eyebrow">Connection interrupted</p>
            <h2>The race archive is unavailable.</h2>
            <p>{dashboardState.message}</p>
            <button type="button" onClick={retry}>
              Try again
            </button>
          </div>
        )}

        {dashboardState.kind === 'loaded' &&
          dashboardState.dashboard.races.length === 0 && (
            <div className="dashboard-message">
              <p className="eyebrow">Archive empty</p>
              <h2>No race weekends are published yet.</h2>
              <p>Complete classifications will appear here after publication.</p>
            </div>
          )}

        {dashboardState.kind === 'loaded' &&
          dashboardState.dashboard.races.length > 0 && (
            <div>
              <div className="section-heading">
                <div>
                  <p className="eyebrow">Complete weekends</p>
                  <h2>Published races</h2>
                </div>
                <p>{dashboardState.dashboard.races.length} classified</p>
              </div>

              <div className="race-grid">
                {dashboardState.dashboard.races.map((race) => (
                  <article className="race-card" key={race.id}>
                    <div className="race-card__season" aria-label={`Season ${race.season}`}>
                      {race.season}
                    </div>
                    <p className="race-card__location">
                      {race.circuit.location} / {race.circuit.countryCode}
                    </p>
                    <h3>
                      <a href={`/races/${encodeURIComponent(race.id)}`}>
                        {race.name}
                      </a>
                    </h3>
                    <p className="race-card__official-name">{race.officialName}</p>
                    <dl className="race-card__facts">
                      <div>
                        <dt>Circuit</dt>
                        <dd>{race.circuit.name}</dd>
                      </div>
                      <div>
                        <dt>Weekend</dt>
                        <dd>
                          <time dateTime={race.startAt}>
                            {dateFormatter.format(new Date(race.startAt))}
                          </time>
                          {' - '}
                          <time dateTime={race.endAt}>
                            {dateFormatter.format(new Date(race.endAt))}
                          </time>
                        </dd>
                      </div>
                    </dl>
                    <footer className="race-card__publication">
                      <span className="complete-mark">Complete</span>
                      <span>
                        Source updated{' '}
                        <time dateTime={race.coverage.sourceFetchedAt}>
                          {dateTimeFormatter.format(
                            new Date(race.coverage.sourceFetchedAt),
                          )}
                        </time>
                      </span>
                      <span>
                        Published{' '}
                        <time dateTime={race.coverage.publishedAt}>
                          {dateTimeFormatter.format(
                            new Date(race.coverage.publishedAt),
                          )}
                        </time>
                      </span>
                    </footer>
                  </article>
                ))}
              </div>
            </div>
          )}
      </section>
    </main>
  )
}
