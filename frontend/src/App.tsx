import { useEffect, useState } from 'react'
import { getReadiness } from './api/client'

type AppProps = {
  loadReadiness?: typeof getReadiness
}

type SystemState =
  | { kind: 'loading' }
  | { kind: 'ready' }
  | { kind: 'unavailable'; message: string }

export default function App({ loadReadiness = getReadiness }: AppProps) {
  const [attempt, setAttempt] = useState(0)
  const [systemState, setSystemState] = useState<SystemState>({
    kind: 'loading',
  })

  useEffect(() => {
    const controller = new AbortController()
    let active = true

    loadReadiness({ signal: controller.signal }).then(
      () => {
        if (active) setSystemState({ kind: 'ready' })
      },
      (error: unknown) => {
        if (active) {
          setSystemState({
            kind: 'unavailable',
            message:
              error instanceof Error
                ? error.message
                : 'The readiness check failed.',
          })
        }
      },
    )

    return () => {
      active = false
      controller.abort()
    }
  }, [attempt, loadReadiness])

  function retry() {
    setSystemState({ kind: 'loading' })
    setAttempt((currentAttempt) => currentAttempt + 1)
  }

  return (
    <main>
      <header className="hero">
        <p className="eyebrow">RaceScope / System status</p>
        <h1>Race intelligence, from lights out to the last lap.</h1>
        <p className="intro">
          A focused home for Formula 1 schedules, results, standings, and race
          analysis.
        </p>
      </header>

      <section
        className={`system-status system-status--${systemState.kind}`}
        role={systemState.kind === 'unavailable' ? 'alert' : 'status'}
        aria-live="polite"
        aria-busy={systemState.kind === 'loading'}
      >
        <div className="status-label">
          <span className="status-light" aria-hidden="true" />
          <span>API / Database</span>
        </div>

        {systemState.kind === 'loading' && (
          <>
            <h2>Checking the pit wall.</h2>
            <p>Confirming that the API and race database are responding.</p>
          </>
        )}

        {systemState.kind === 'ready' && (
          <>
            <h2>Systems are ready.</h2>
            <p>The API is running and connected to PostgreSQL.</p>
          </>
        )}

        {systemState.kind === 'unavailable' && (
          <>
            <h2>RaceScope is unavailable.</h2>
            <p>{systemState.message}</p>
            <button type="button" onClick={retry}>
              Try again
            </button>
          </>
        )}
      </section>
    </main>
  )
}
