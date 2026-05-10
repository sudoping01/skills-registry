# SkillHub CLI

> The official CLI for [SkillHub](https://github.com/sudoping01/skillhub) — the open-source registry for AI agent skills.

Install, publish, search, and manage SKILL.md bundles from your terminal.

---

## Repos

| Repo | Purpose |
|---|---|
| **[sudoping01/skills-registry](https://github.com/sudoping01/skills-registry)** | This repo — CLI + standalone server |
| **[sudoping01/skillhub](https://github.com/sudoping01/skillhub)** | SkillHub server (Forgejo fork) — self-host the full registry |

---

## Install

```bash
git clone https://github.com/sudoping01/skills-registry
cd skills-registry
make install        # builds and copies skill to /usr/local/bin
skill --help
```

Or build without installing:

```bash
make build          # → ./bin/skill
```

---

## CLI Commands

```bash
skill init          # scaffold a new skill directory interactively
skill validate .    # validate a skill and get a quality score (0-100)
skill push .        # publish to the registry
skill pull user/name           # fetch just the SKILL.md (no install)
skill install user/name        # install a skill + all its dependencies
skill install user/name@1.2.0  # install a specific version
skill update .      # bump patch version and re-publish
skill search "pdf extraction"              # search the registry
skill search --license MIT --sort stars    # filter results
skill search --compat Claude --sort downloads
skill info user/name    # show skill metadata
skill login             # save your API token locally
```

---

## Skill Format

A skill is a directory with a `SKILL.md` at the root:

```
my-skill/
├── SKILL.md          ← required
├── scripts/          ← optional
├── references/       ← optional
└── assets/           ← optional
```

`SKILL.md` starts with YAML frontmatter:

```markdown
---
name: "web-scraper"
description: "Scrape structured data from any webpage."
license: "MIT"
compatibility: "Claude Code, Claude Sonnet 4"
metadata:
  author: "sudoping01"
  version: "1.0.0"
dependencies:
  - "sudoping01/html-parser"
  - "sudoping01/file-writer@1.2.0"
---

# Web Scraper

## Overview
...
```

**Validation rules:**
- `name` — required; lowercase + digits + hyphens; 1–64 chars; must match directory name
- `description` — required; max 1024 chars; should start with a verb
- `compatibility` — optional; max 500 chars
- `dependencies` — optional; list of `user/name` or `user/name@version`
- Bundle max size: 2 MB

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SKILLHUB_REGISTRY` | `http://localhost:3000` | Registry URL |
| `SKILLHUB_USER` | — | Your username (required for push/update) |
| `SKILLHUB_TOKEN` | — | API token (or use `skill login`) |

---

## Self-hosting

The standalone server in `server/` is a lightweight alternative to the full Forgejo-based registry.
For the full-featured self-hosted registry with user accounts, stars, and a web UI, use the
**[SkillHub server](https://github.com/sudoping01/skillhub)**.

```bash
# Standalone server (no user accounts, simple token auth)
make build
SKILLHUB_PORT=3000 ./bin/skillhub-server

# Full registry (Docker)
git clone https://github.com/sudoping01/skillhub
cd skillhub
make up
```

---

## API Reference

The CLI talks to the SkillHub API. All endpoints are prefixed with `/api/skillhub/v1`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/search?q=&license=&compat=&sort=` | — | Search skills |
| `GET` | `/skills/{user}/{name}` | — | Get skill metadata |
| `GET` | `/skills/{user}/{name}/download` | — | Download `.skill` archive |
| `GET` | `/skills/{user}/{name}/readme` | — | Fetch raw SKILL.md |
| `GET` | `/skills/{user}/{name}/badge` | — | SVG badge |
| `POST` | `/skills/{user}/{name}` | Bearer token | Publish a skill |
| `PUT` | `/skills/{user}/{name}/star` | Bearer token | Star a skill |
| `DELETE` | `/skills/{user}/{name}/star` | Bearer token | Unstar a skill |

---

## Publishing with CI/CD

Tag a release and let GitHub Actions publish automatically.

Copy `.github/workflows/publish-skill.yml` from
[contrib/github-publish-skill.yml](https://github.com/sudoping01/skills-registry/blob/main/contrib/github-publish-skill.yml)
into your skill repo and add two secrets:
- `SKILLHUB_TOKEN` — your API token
- `SKILLHUB_REGISTRY` — your SkillHub instance URL

Then push a tag:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

---

## Development

```bash
make build          # build CLI binary
make test           # run all tests
make docker-build   # build CLI Docker image
make release        # cross-platform binaries → dist/
```

---

## License

GPL-3.0-or-later
