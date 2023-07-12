set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# pushes all changes to the main branch
push +COMMIT_MESSAGE:
  git add .
  git commit -m "{{COMMIT_MESSAGE}}"
  git pull origin main
  git push origin main

# runs the ent generator
generate:
  cd api && go generate ./ent

# creates a new ent entity
new-entity NAME:
  cd api && go run -mod=mod entgo.io/ent/cmd/ent new {{NAME}}

# stop the development environment
stop:
  docker stop shutterbase-db
  docker stop shutterbase-s3

alias b := backend
# start the development environment
backend:
  ./hack/start-postgres.sh
  ./hack/start-minio.sh
  echo waiting for postgres and minio startup
  sleep 10
  # cd api && go run internal/cmd/user-seed/main.go
  # cd api && go run .