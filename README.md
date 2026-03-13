# JulesCord 🤖

A Discord bot built entirely by [Jules](https://jules.google.com) — Google's autonomous coding agent — with zero human code contributions.

Every 15 minutes, Jules reads its own state file (`AGENTS.md`), picks the next task, implements it, and opens a PR. The PR gets auto-merged. Repeat.

## How it works

```
GitHub Actions cron (every 15min)
        ↓
  jules-invoke action
        ↓
  Jules reads AGENTS.md
        ↓
  Jules implements next feature
        ↓
  Jules updates AGENTS.md + opens PR
        ↓
  auto-merge workflow merges PR
        ↓
        🔁
```

## Setup

To run this yourself you need:
- A Jules API key (from [jules.google.com](https://jules.google.com) → Settings)
- The Jules GitHub app installed on this repo
- `JULES_API_KEY` added as a GitHub Actions secret

## Documentation

JulesCord is a complex application with a Go backend, PostgreSQL database, and a React web dashboard. The documentation is split into several files in the `docs/` directory:

- [Setup Guide](docs/setup.md)
- [Architecture](docs/architecture.md)
- [Bot Features](docs/features.md)
- [Available Commands](docs/commands.md)
- [API Documentation](docs/api.md)

## What Jules is building

A Discord bot themed around itself. See `AGENTS.md` for current progress.
