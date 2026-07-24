# Tracer Bullet Planner

Convert an approved implementation plan into small, ordered tasks using the
tracer bullet development method.

## Inputs

- Read the project plan provided by the user.
- Read relevant architecture and project-convention files.
- Do not change architectural decisions unless the plan is impossible or
  contradictory.
- Ask questions only when a missing decision blocks implementation.

## Goal

Produce a tasks file that builds thin, end-to-end working paths through the
application.

A tracer bullet should cross the necessary layers, such as:

- database or external data source
- repository or data-access layer
- business logic
- API endpoint
- frontend API client
- user interface
- automated tests

Do not build an entire layer before connecting it to the rest of the system.

## Task rules

Each task must:

1. Be independently understandable.
2. Be small enough for one focused coding session.
3. Produce a testable or observable result.
4. Name the files or areas likely to change.
5. Include clear completion criteria.
6. Avoid unrelated cleanup or speculative abstractions.
7. Preserve a working application whenever practical.

## Output structure

Create `tasks/current.md` using this structure:

# Current Implementation Tasks

## Tracer Bullet 1: [Working capability]

### Goal

Describe the smallest end-to-end behavior this tracer bullet proves.

### Task 1: [Task title]

**Purpose:** Why this task is needed.

**Likely files:**

- `path/to/file`
- `path/to/other-file`

**Steps:**

- [ ] Small implementation step
- [ ] Small implementation step
- [ ] Add or update tests
- [ ] Verify the completion criteria

**Complete when:**

- Concrete observable result
- Relevant tests pass
- Type checking passes

### Task 2: [Task title]

Continue in dependency order.

## Tracer Bullet 2: [Next working capability]

Only include this section when it logically follows from the first tracer
bullet.

## Validation

- [ ] Backend tests pass
- [ ] Frontend tests pass
- [ ] Type checking passes
- [ ] Database migrations run successfully
- [ ] The end-to-end behavior works manually

## Restrictions

- Do not implement the tasks.
- Do not rewrite the original plan.
- Do not generate hundreds of tiny mechanical tasks.
- Do not combine multiple major features into one task.
- Do not add features that are not required by the plan.