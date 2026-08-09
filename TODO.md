# TODOs for FSG

## Bad

- close tagging dialog when clicking outside of it
- multiselect in grid view
- filter for upload with query parameter, etc

## Not so bad

- delete cameras
- add pagination or infiniscroll to project tags
- tag ui broken when tag description too long
- applied tag shows up locally as "manual" even though it might be "custom"
- verify S3 completeness after Hetzner transfer finishes: compare bucket originals (`XX/<storageId>.jpg`) against the 29108 `images.storage_id` rows in prod (mc ls -r diff or `import --verify` re-run)
- ui: `vue-tsc --noEmit` crashes ("isSatisfiesExpression is not defined") — vue-tsc/@vue/language-core incompatible with the pinned typescript; templates currently have no typecheck gate. Bump vue-tsc + @vue/language-core (or typescript) until it runs clean
- ci: `Server Build` job always fails on fork PRs — the docker push step runs with the fork's read-only GITHUB_TOKEN ("denied: installation not allowed to Write organization package"). Guard the push with `if: github.event_name != 'pull_request'` (build-only for PRs) so fork contributions can go green
- schema hardening: all other `name` fields (upload, imageTag, personName, camera, project, apiKey) are still unlimited varchar — same class of bug as the download-config name (#90). Before capping them, check prod for existing rows longer than the target limit; an ent auto-migrate column shrink fails on oversized rows and stalls the rollout
