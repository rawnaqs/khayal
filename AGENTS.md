# Agents

## Agent skills

### Issue tracker

Issues live in GitHub. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical labels in use. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout: one `CONTEXT.md` at repo root, ADRs in `docs/adr/`. See `docs/agents/domain.md`.

### Standing rules

- **Keep `testdata/config.yaml` in sync**: whenever config surface
  changes (new fields, toggles, defaults), update
  `testdata/config.yaml`, `config.example.yaml`, and the SPEC config
  section together. The dev-server verification flow runs against
  testdata — stale configs there cause misleading live tests.
