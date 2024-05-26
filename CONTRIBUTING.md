# Contributing to Shutterbase

Thank you for your interest in contributing to Shutterbase! We welcome contributions from the community and are excited to see how you can help improve our project. To ensure a smooth collaboration, please follow these guidelines.

## Table of Contents
1. [Getting Started](#getting-started)
2. [Branch Naming](#branch-naming)
3. [Commit Messages](#commit-messages)
4. [Creating a Pull Request](#creating-a-pull-request)
5. [Maintaining Linear History](#maintaining-linear-history)
6. [Rebasing and Force Pushing](#rebasing-and-force-pushing)

## Getting Started

1. Fork the repository on GitHub.
2. Clone your fork to your local machine:
   ```sh
   git clone https://github.com/your-username/shutterbase.git
   ```
3. Navigate to the project directory:
   ```sh
   cd shutterbase
   ```
4. Add the original repository as a remote to keep your fork updated:
   ```sh
   git remote add upstream https://github.com/shutterbase/shutterbase.git
   ```

## Branch Naming

We follow the Gitflow workflow for our branch naming conventions. Please create branches using the following prefixes:

- `feature/xxx` - for new features
- `fix/xxx` - for bug fixes
- `docs/xxx` - for documentation changes
- `chore/xxx` - for maintenance tasks
- `refactor/xxx` - for code refactoring

Example:
```sh
git checkout -b feature/add-new-feature
```

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) for our commit messages. This makes it easier to understand the history of changes and automate the release process.

Format:
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat` - a new feature
- `fix` - a bug fix
- `docs` - documentation changes
- `style` - changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- `refactor` - code change that neither fixes a bug nor adds a feature
- `test` - adding or correcting tests
- `chore` - changes to the build process or auxiliary tools and libraries such as documentation generation

Example:
```
feat(auth): add OAuth2 login
```

## Creating a Pull Request

1. Ensure all tests pass locally.
2. Push your branch to your fork:
   ```sh
   git push origin feature/your-branch-name
   ```
3. Open a pull request (PR) on GitHub.
4. Follow the PR template and fill out all required fields.
5. Ensure the PR description clearly explains the purpose and details of the changes.

All work should be done on respective branches and be merged into `main` using a PR on GitHub. The `main` branch is protected against direct pushes, and linear history is enforced.

## Maintaining Linear History

To keep our project history clean and readable, we enforce a linear history on the `main` branch. This means using rebase instead of merge when integrating changes from `main` into your feature branch.

### Rebasing and Force Pushing

If your branch gets out of sync with `main`, you need to rebase it. Here's how to do it:

1. Fetch the latest changes from the upstream repository:
   ```sh
   git fetch upstream
   ```
2. Checkout your feature branch:
   ```sh
   git checkout feature/your-branch-name
   ```
3. Rebase your branch onto the latest `main`:
   ```sh
   git rebase upstream/main
   ```
4. If you encounter conflicts, resolve them and continue the rebase:
   ```sh
   git add .
   git rebase --continue
   ```
5. After a successful rebase, force push your changes:
   ```sh
   git push --force-with-lease
   ```

Example:
```sh
git fetch upstream
git checkout feature/add-new-feature
git rebase upstream/main
# Resolve any conflicts, then:
git add .
git rebase --continue
git push --force-with-lease
```

By following these guidelines, you help us maintain a clean and efficient workflow. Thank you for contributing to Shutterbase!
