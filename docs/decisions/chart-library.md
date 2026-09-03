# Chart Library Selection

**Status:** Selected for implementation
**Decision date:** 2026-08-12
**Selected library:** Recharts 3

## Context

The first production chart compares exactly two drivers over a Grand Prix. The real
`POST /api/statistics/query` response supplies every source lap, nullable integer
microsecond durations, compound and stint context, sourced boundary markers, and
canonical colour, line-style, and marker choices. A representative race has roughly
50-80 laps per driver, so customizable SVG is appropriate.

The chart is a visual summary. A complete semantic table will remain the authoritative
accessible form of every observation, rather than placing every plotted lap in the tab
order.

## Evaluation

| Requirement | Recharts 3 | Visx 4 | Nivo 0.99 | Apache ECharts 6 |
| --- | --- | --- | --- | --- |
| React and responsive SVG | Native | Native, lower-level | Native | Adapter and explicit resize handling |
| Null gaps and custom styles | Supported | Supported | Supported | Supported |
| Keyboard tooltip navigation | Built-in chart accessibility layer | Application-owned | No comparable evidence found | Application-owned |
| Accessible title/description | Chart API | Application-owned SVG | Application-owned | ARIA option model |
| License | MIT | MIT | MIT | Apache-2.0 |

## Decision

Use **Recharts 3**. It provides responsive native SVG, React composition, null gaps,
custom lines and markers, and one chart-level keyboard target with arrow-key tooltip
navigation. Visx offers more direct SVG control but requires more accessibility work.
Nivo offers no stronger accessibility evidence for this requirement. ECharts is capable
but its imperative integration and larger package surface are unnecessary here.

## Implementation Constraints

- Keep numeric laps and durations in source units until display formatting.
- Leave null durations disconnected and never interpolate them.
- Use API-provided colour, line style, and marker redundantly.
- Provide an accessible title, description, visible focus, and keyboard instructions.
- Keep one chart focus target, not one target per observation.
- Disable nonessential animation and retain a complete semantic table.

## Evidence

- [Recharts package metadata](https://www.npmjs.com/package/recharts)
- [Recharts LineChart API](https://recharts.github.io/en-US/api/LineChart/)
- [Visx XYChart](https://www.npmjs.com/package/@visx/xychart)
- [Nivo line](https://www.npmjs.com/package/@nivo/line)
- [Apache ECharts accessibility](https://echarts.apache.org/handbook/en/best-practices/aria/)
