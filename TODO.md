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
