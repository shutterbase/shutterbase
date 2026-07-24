---
description: Roll a released version out to the FSG cluster via the argocd-fsg-web GitOps repo
argument-hint: "[vX.Y.Z] (optional — defaults to the newest tag)"
allowed-tools: Bash, Read, Edit, Grep, Glob
---

Deploy a **released** shutterbase version to FSG prod.

Requested version (may be empty — use the newest `v*` tag then): $ARGUMENTS

## What this deployment is

- The cluster is GitOps-managed: **ArgoCD watches the repo, nothing is applied by hand.**
  A push to `master` of `argocd-fsg-web` *is* the deploy.
- GitOps repo: `argocd-fsg-web` on the FSG GitLab
  (`gitlab.fs-g.org/FormulaStudentGermany/web/argocd-fsg-web`), branch **`master`** — not
  `main`. Local checkout: `~/gitlab.fs-g.org/FormulaStudentGermany/web/argocd-fsg-web`.
  If it is missing, stop and ask — do not clone a guess.
- Manifest: `shutterbase/deployment.yml`, namespace `fsgshutterbase`, 2 replicas,
  RollingUpdate `maxSurge: 1` / `maxUnavailable: 0`.
- All credentials come from Vault at runtime (`DATABASE_CREDENTIALS_SOURCE=vault`,
  `S3_CREDENTIALS_SOURCE=vault`, `VAULT_ENV_KV_PATH`). There are no SealedSecrets left in
  this deployment — a version bump touches nothing but the tag.

## Steps

### 1. Resolve the version

Use the argument, else the newest tag from the app repo:

```bash
git -C ~/github.com/shutterbase/shutterbase tag --sort=-v:refname | head -1
```

### 2. Refuse to deploy an image that does not exist

The tag-triggered CI is what publishes the image, and it finishes *after* the tag push.
Deploying ahead of it gives you an ImagePullBackOff on prod.

```bash
docker manifest inspect ghcr.io/shutterbase/shutterbase:vX.Y.Z > /dev/null
```

If that fails, stop. Check whether the release run is still going
(`gh run list --repo shutterbase/shutterbase --limit 3`) and wait for it rather than
pushing a manifest that cannot pull.

### 3. Bump the tag — in BOTH places

`shutterbase/deployment.yml` carries the version **twice**, and a deploy that updates only
the first one lies to the running app about which version it is:

```yaml
image: ghcr.io/shutterbase/shutterbase:vX.Y.Z   # what actually runs
- name: DEPLOYMENT_IMAGE_TAG
  value: vX.Y.Z                                  # what the app reports about itself
```

After editing, confirm no stale version remains:

```bash
grep -n "shutterbase:v\|DEPLOYMENT_IMAGE_TAG" -A1 shutterbase/deployment.yml
```

### 4. Check whether the release needs more than a tag bump

Read the release notes / diff since the currently deployed version. If it introduced a new
config key, a new Vault path, a new secret, or a schema change needing coordination, the
manifest needs that change **in the same commit** — a tag-only bump would roll out a pod
that crash-loops on missing config. When in doubt, say what you found and ask.

Schema migrations themselves need no action: ent auto-migrate runs on startup.

### 5. Commit and push (this is the deploy)

The repo has its own recipe:

```bash
cd ~/gitlab.fs-g.org/FormulaStudentGermany/web/argocd-fsg-web
just push "chore(shutterbase): deploy vX.Y.Z"
```

(equivalently: `git add . && git commit && git pull origin master && git push origin master`)

**Confirm with the user before this push.** It is an immediate, irreversible rollout to
production — ArgoCD picks it up on its own.

Push only the shutterbase change. If `git status` shows unrelated dirty files from other
apps in that repo, stop and ask — `just push` stages everything with `git add .`.

### 6. Verify the rollout against the running pods

Do **not** poll ArgoCD — there is no FSG argocd context on this machine and the CLI
points at an unrelated cluster. The app reports its own deployed tag, which is a stronger
signal anyway: ArgoCD "Synced" only means the manifest was applied, not that pods are
serving the new image.

`/api/v1/health` returns `DEPLOYMENT_IMAGE_TAG` — the value this very manifest sets — so
it flips to the new version only once a new pod is actually answering:

```bash
curl -s https://shutterbase.fsg.one/api/v1/health
# {"status":"ok","version":"vX.Y.Z"}
```

Poll until it matches, then stop. ArgoCD's sync interval plus a rolling update means a
few minutes is normal:

```bash
for i in $(seq 1 40); do
  v=$(curl -s --max-time 15 https://shutterbase.fsg.one/api/v1/health)
  echo "$i: $v"
  case "$v" in *'"version":"vX.Y.Z"'*) echo "rolled"; break;; esac
  sleep 20
done
```

Two replicas roll one at a time (`maxSurge 1` / `maxUnavailable 0`), so during the roll the
endpoint may answer from either the old or the new pod — treat a stable new version across
a couple of consecutive polls as done, not the first hit.

If it never flips, the deploy did not reach the pods. Check, in order: the GitOps commit is
on `master`; the ArgoCD app synced (ask the user to look in the UI — no CLI session here);
the image tag exists on ghcr; the pod is not in ImagePullBackOff. `maxUnavailable: 0` means
a failed pull never degrades the service — the rollout simply never completes, and the old
version keeps serving.

### 7. Report

Version deployed, the GitOps commit, and what `/api/v1/health` reports. If the version
never flipped, say so plainly rather than calling the deploy done — the GitOps push
succeeding is not the same as prod running the new binary.
