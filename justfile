set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# pushes all changes to the main branch
push +COMMIT_MESSAGE:
  git add .
  git commit -m "{{COMMIT_MESSAGE}}"
  git pull origin main
  git push origin main
