# SkillHub

An open-source registry and CLI for AI agent skills — like npm, but for SKILL.md bundles.

## Quick Start

### Build

```bash
make build
```

### Run the registry

```bash
make run-server
```

### Use the CLI

```bash
# Validate a skill
./bin/skill validate ./my-skill

# Publish
SKILLHUB_USER=sudoping01 ./bin/skill push ./my-skill

# Install
./bin/skill install sudoping01/my-skill

# Install a specific version
./bin/skill install sudoping01/my-skill@1.2.0

# Search
./bin/skill search "pdf extraction"

# Info
./bin/skill info sudoping01/my-skill
```

## Skill Format

A skill is a directory with a `SKILL.md` file at the root:

```
my-skill/
├── SKILL.md          ← Required
├── scripts/          ← Optional
├── references/       ← Optional
└── assets/           ← Optional
```

`SKILL.md` starts with YAML frontmatter:

```markdown
---
name: "my-skill"
description: "Publish datasets to Hugging Face Hub. Use when uploading datasets."
license: "Apache-2.0"
compatibility: "Tested with Python 3.8+"
metadata:
  author: "ml-team"
  version: "1.0.0"
---

# My Skill
...
```

**Validation rules:**
- `name` — required; lowercase, digits, hyphens; 1–64 chars; must match directory name
- `description` — required; max 1024 chars
- `compatibility` — optional; max 500 chars
- Total bundle size — max 2 MB

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SKILLHUB_REGISTRY` | `http://localhost:8080` | Registry URL |
| `SKILLHUB_USER` | — | Your username (for push) |
| `SKILLHUB_TOKEN` | — | Auth token (optional; enables token auth on server) |
| `SKILLHUB_DATA_DIR` | `./data` | Server data directory |
| `SKILLHUB_PORT` | `8080` | Server port |

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/search?q=<query>` | Search skills |
| `GET` | `/api/v1/skills/{user}/{name}/info` | Get skill metadata |
| `GET` | `/api/v1/skills/{user}/{name}/download?version=<v>` | Download .skill archive |
| `POST` | `/api/v1/skills/{user}/{name}` | Publish a skill |

## License

GPL v3
