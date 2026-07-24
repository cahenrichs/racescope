# RaceScope Product Vision

## Objective

Build a portfolio-quality Formula 1 statistics platform that makes core information approachable to casual viewers and progressively exposes deeper analysis.

Public V1 will cover OpenF1 data from 2023 onward. Users will be able to:

- See the next race, previous race result, and current championship leaders.
- Browse schedules, race details, results, drivers, and standings by season.
- Run three supported statistical analyses with shareable URLs.
- Use the core experience on modern mobile and desktop devices.

The work will begin with an internal tracer-bullet MVP before expanding to the public V1. See the [V1 plan](v1-plan.md) for its committed scope and delivery milestones.

## Product Decision Rules

- V1 serves casual fans, statistics enthusiasts, and portfolio reviewers. When their needs conflict, decide per feature rather than applying one global audience priority.
- Before implementation, label each feature with one primary audience, one measurable user outcome, and a fixed scope.
- Anchor feature outcomes to three product tasks: casual fans can find the next race, latest podium, and leaders unaided; enthusiasts can configure, understand, and share each analysis; reviewers can run the project and trace its architecture and data decisions from documentation.
- There is no fixed launch deadline. Milestone 3 is the single planned scope-revision gate; after its review, freeze Public V1 scope until release.

## Post-V1 Direction

Immediately after launch, add an architecture and data-flow diagram, a tradeoff narrative, polished API documentation, screenshots or demo media, and a portfolio case study. Deeper race analysis, additional metrics, telemetry, teammate comparisons, pre-2023 history, and additional providers remain separate future milestones.

Post-V1 analysis extensions already identified during planning are selectable lap filters for pit, neutralized, and anomalous laps; selectable point modes for weekend total, Grand Prix only, and separate sprint/Grand Prix events; and two-driver season or full-field single-race modes for grid-versus-finish analysis.
