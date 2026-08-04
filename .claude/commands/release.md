---
description: Ship the working tree as a release — branch, PR, watch CI, merge, tag, watch the release run
argument-hint: "[patch|minor|major] (optional — inferred from the changes when omitted)"
allowed-tools: Bash, Read, Edit, Write, Grep, Glob
---

Take everything currently in the working tree and carry it all the way to a published
release of `shutterbase/shutterbase`.

Requested bump (may be empty — infer it then): $ARGUMENTS

## What the release machinery actually is

- `main` is protected and enforces **linear history**. Direct pushes are rejected; every
  change lands through a PR, merged with **squash** or **rebase** (never a merge commit).
- **A release is a tag.** `.github/workflows/build.yml` runs its tag path only for
  `refs/tags/v*.*.*`. Pushing `vX.Y.Z` creates the GitHub release, uploads the `downloader`
  binaries for 6 os/arch pairs, and publishes `ghcr.io/shutterbase/shutterbase:vX.Y.Z`.
  There is no separate release workflow file.
- **The tag run does not rebuild.** It re-tags: the `retag` job stamps `:vX.Y.Z` onto the
  `:<sha8>` manifest the main-branch run already built and tested, so the released image is
  the exact digest CI proved green. Tests and the image build are skipped on tags. The tag
  run therefore finishes in ~1 minute, not ~6 — but it **depends on the main-branch build
  for the merge commit having succeeded**. `retag` polls the registry for up to 20 minutes,
  so tagging straight after the merge is fine; a main build that failed is not.
- PR CI = two workflows: **Build and Release** (`test-api`, `test-ui`, `downloader-build`,
  `server`) and **UI E2E** (Playwright). Both must be green.
- Commits follow **Conventional Commits** — the changelog is generated from them.

## Steps

### 1. Survey before touching anything

```bash
git status --short
git diff --stat
git log --oneline -5
git tag --sort=-v:refname | head -3
```

Read the actual diff, not just the file list. You need to know what changed to write the
commit message and pick the bump.

Stop and report instead of proceeding if:
- the working tree is clean (nothing to release), or
- you are not on `main`, or the branch is not up to date with `origin/main`.

### 2. Pick the version

Base it on the highest-ranking change in the diff, not on the file count:

| Change | Bump |
|---|---|
| `feat:` — new capability, new endpoint, new UI surface | **minor** |
| `fix:`, `refactor:`, `chore:`, `docs:`, `test:`, `style:` | **patch** |
| Breaking API/schema change, removed config key, anything needing a coordinated deploy | **major** — ASK first, never decide this alone |

An explicit argument overrides the inference. Compute the next version from the newest
`v*` tag.

### 3. Verify locally before spending CI minutes

```bash
cd api && just test-unit && just test-e2e   # e2e needs Docker (testcontainers)
cd ui && bun run test && bun run build
```

Fix anything red **before** opening the PR. A red PR is slower than a red laptop.

### 4. Branch, commit, push

Branch prefix must match the change: `feature/`, `fix/`, `docs/`, `chore/`, `refactor/`.

```bash
git checkout -b <prefix>/<short-kebab-summary>
git add -A
git commit -m "<type>(<scope>): <subject>"   # Conventional Commits; body for the why
git push -u origin HEAD
```

Never add a `Co-Authored-By` line.

### 5. Open the PR and watch CI

```bash
gh pr create --fill --base main
gh pr checks --watch
```

**If a check fails:** read the failing job's log (`gh run view <id> --log-failed`), fix the
cause in the working tree, commit, push, and watch again. Repeat until green. Fix the
actual defect — never disable a test or weaken an assertion to get a green tick. If a
failure is genuinely infrastructure flake (runner networking, container pull), say so
explicitly and re-run that job rather than pretending it passed.

### 6. Confirm, then merge

This is the point of no return: merging touches protected `main` and the tag that follows
publishes a public release and image. **Report the plan and get a yes** before continuing:
the version, the PR link, the CI result, and a one-line summary of what ships.

Then merge. `main` requires one approving review and MaPa is the sole maintainer —
GitHub forbids approving your own PR, so a self-authored release PR is always `BLOCKED` /
`REVIEW_REQUIRED`. `enforce_admins` is false and the admin bypass is the intended path
here; use it without asking again (the confirmation above is about publishing the version,
not about the review rule):

```bash
gh pr merge --squash --admin --delete-branch  # squash keeps main linear
git checkout main && git pull --ff-only origin main
```

### 7. Tag and watch the release run

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
gh run watch --exit-status                     # the tag-triggered Build and Release run
```

Watch it to completion. If the release run fails after the tag is already pushed, say so
plainly — the tag exists but the release may be partial. The usual failure is `retag`
timing out because the main-branch build for the merge commit never produced
`:<sha8>`; fix that build and re-run the job rather than retagging. Do not silently
move an existing tag.

### 8. Verify the artifact exists

```bash
gh release view vX.Y.Z
docker manifest inspect ghcr.io/shutterbase/shutterbase:vX.Y.Z > /dev/null && echo "image published"
```

### 9. Report

State the version, the PR, the release URL, whether the image is on ghcr, and anything
that had to be fixed on the way. Then mention that `/deploy-fsg` rolls this version out to
the FSG cluster — do **not** deploy as part of this command.
