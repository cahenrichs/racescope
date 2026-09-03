import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type DotItemDotProps,
} from 'recharts'
import type { LapComparisonResult, LapSeries } from '../../api/contracts'

type LapComparisonChartProps = {
  result: LapComparisonResult
}

type ChartRow = {
  lapNumber: number
  [driverId: string]: number | null
}

export default function LapComparisonChart({ result }: LapComparisonChartProps) {
  const data = buildChartData(result.series)
  const description = `${result.title}. Lap number is shown on the horizontal axis and lap duration on the vertical axis. Missing durations appear as gaps.`

  return (
    <figure className="lap-chart" aria-labelledby="lap-chart-title">
      <figcaption>
        <p className="eyebrow">Lap trace</p>
        <h2 id="lap-chart-title">{result.title}</h2>
        <p id="lap-chart-description">{description}</p>
        <p id="lap-chart-instructions" className="lap-chart__instructions">
          Focus the chart, then use the left and right arrow keys to inspect each lap. Press Enter to
          keep or dismiss the current tooltip.
        </p>
      </figcaption>
      <ul className="lap-chart__legend" aria-label="Driver chart styles">
        {result.series.map((series) => (
          <li key={series.driver.id}>
            <svg width="44" height="12" aria-hidden="true">
              <line
                x1="1"
                x2="43"
                y1="6"
                y2="6"
                stroke={series.style.colour}
                strokeDasharray={dashPattern(series.style.lineStyle)}
                strokeWidth="3"
              />
              {series.style.marker === 'square' ? (
                <rect x="18" y="2" width="8" height="8" fill={series.style.colour} />
              ) : (
                <circle cx="22" cy="6" r="4" fill={series.style.colour} />
              )}
            </svg>
            <span>
              {series.driver.name}, {series.style.lineStyle} line with {series.style.marker} markers
            </span>
          </li>
        ))}
      </ul>
      <div className="lap-chart__plot">
        <ResponsiveContainer
          width="100%"
          height="100%"
          initialDimension={{ width: 960, height: 420 }}
        >
          <LineChart
            data={data}
            margin={{ top: 16, right: 16, bottom: 16, left: 8 }}
            accessibilityLayer
            aria-describedby="lap-chart-description lap-chart-instructions"
            aria-keyshortcuts="ArrowLeft ArrowRight Enter"
            role="application"
            title={result.title}
            desc={description}
          >
            <CartesianGrid stroke="#35383d" strokeDasharray="3 5" vertical={false} />
            <XAxis
              dataKey="lapNumber"
              type="number"
              domain={['dataMin', 'dataMax']}
              tick={{ fill: '#b9bbc0' }}
              tickLine={false}
              label={{ value: 'Lap', position: 'insideBottom', fill: '#b9bbc0', offset: -8 }}
            />
            <YAxis
              domain={['auto', 'auto']}
              tick={{ fill: '#b9bbc0' }}
              tickFormatter={formatAxisDuration}
              tickLine={false}
              width={68}
              label={{ value: 'Lap time', angle: -90, position: 'insideLeft', fill: '#b9bbc0' }}
            />
            <Tooltip
              animationDuration={0}
              formatter={(value, driverId) => [
                formatDuration(Number(value)),
                result.series.find((series) => series.driver.id === driverId)?.driver.name ??
                  driverId,
              ]}
              labelFormatter={(lap) => `Lap ${lap}`}
            />
            {result.series.map((series) => (
              <Line
                key={series.driver.id}
                dataKey={series.driver.id}
                name={series.driver.id}
                type="linear"
                stroke={series.style.colour}
                strokeDasharray={dashPattern(series.style.lineStyle)}
                strokeWidth={3}
                connectNulls={false}
                isAnimationActive={false}
                dot={(props: DotItemDotProps) => (
                  <SeriesMarker
                    {...props}
                    marker={series.style.marker}
                    colour={series.style.colour}
                  />
                )}
                activeDot={{ r: 6, strokeWidth: 2 }}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </figure>
  )
}

function buildChartData(series: LapSeries[]): ChartRow[] {
  const rows = new Map<number, ChartRow>()

  for (const driverSeries of series) {
    for (const observation of driverSeries.observations) {
      const row = rows.get(observation.lapNumber) ?? { lapNumber: observation.lapNumber }
      row[driverSeries.driver.id] = observation.durationMicroseconds
      rows.set(observation.lapNumber, row)
    }
  }

  return [...rows.values()].sort((a, b) => a.lapNumber - b.lapNumber)
}

function SeriesMarker({
  cx = 0,
  cy = 0,
  marker,
  colour,
}: DotItemDotProps & { marker: LapSeries['style']['marker']; colour: string }) {
  if (marker === 'square') {
    return <rect x={cx - 3} y={cy - 3} width="6" height="6" fill={colour} />
  }
  return <circle cx={cx} cy={cy} r="3" fill={colour} />
}

function dashPattern(lineStyle: LapSeries['style']['lineStyle']) {
  return lineStyle === 'dashed' ? '8 6' : undefined
}

function formatAxisDuration(microseconds: number) {
  return `${(microseconds / 1_000_000).toFixed(0)}s`
}

function formatDuration(microseconds: number) {
  const totalSeconds = microseconds / 1_000_000
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = (totalSeconds % 60).toFixed(3).padStart(6, '0')
  return `${minutes}:${seconds}`
}
