set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# starts local dev dependencies (Postgres + RustFS) and seeds the DB (idempotent)
up:
  docker compose up -d
  docker compose up -d --wait postgres
  cd api && go run ./cmd/seed

# stops local dev dependencies
down:
  docker compose down

# tails logs from local dev dependencies
deps-logs:
  docker compose logs -f

# stops deps and wipes their bind-mounted data under sandbox/
clear: down
  rm -rf sandbox/postgres sandbox/rustfs

# counts lines of code, excluding generated code (ent, except the hand-written schema)
count:
  cloc --vcs=git --fullpath --not-match-d='api/ent/(?!schema)' .

# counts lines of code including generated code
count-all:
  cloc --vcs=git .

# pushes all changes to the main branch
push +COMMIT_MESSAGE:
  git add .
  git commit -m "{{COMMIT_MESSAGE}}"
  git pull origin main
  git push origin main
