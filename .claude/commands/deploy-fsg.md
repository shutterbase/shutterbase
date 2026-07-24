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

### 6. Watch the rollout

There is no FSG kube context configured locally, so verify through ArgoCD (or ask the user
to look):

```bash
argocd app get shutterbase        # needs an argocd login against the FSG instance
```

If no session is available, say so and report what to check in the ArgoCD UI: the
shutterbase app Synced + Healthy, and both replicas on the new tag. RollingUpdate with
`maxUnavailable: 0` means the old pods keep serving until the new ones are ready, so a
failed image pull degrades nothing — it just never completes.

### 7. Verify the running version

```bash
curl -s https://shutterbase.fsg.one/api/v1/version
```

It must report the version you just deployed. If it reports something else, the responding
pod did not come from this manifest — investigate before declaring success (this exact
mismatch has happened before, from a pre-GitOps pod started by hand).

### 8. Report

Version deployed, the GitOps commit, sync/health status, and what `/api/v1/version` says.
