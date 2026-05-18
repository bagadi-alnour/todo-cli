# Release checklist (v0.6.0)

## 1. Tag and push (todo-cli repo)

```bash
cd todo-cli
git tag -a v0.6.0 -m "v0.6.0: per-user storage, assignees, UI improvements"
git push origin v0.6.0
```

GoReleaser will create the GitHub Release from `.github/workflows/release.yml`.

## 2. Verify Homebrew sha256 (after tag is on GitHub)

GitHub’s tarball hash may differ slightly from a local `git archive`. After the tag exists:

```bash
curl -L https://github.com/bagadi-alnour/todo-cli/archive/refs/tags/v0.6.0.tar.gz | shasum -a 256
```

Update `homebrew-tap/Formula/todo.rb` with that sha256 if it differs from the pre-filled value.

## 3. Push homebrew-tap

```bash
cd homebrew-tap
git add Formula/todo.rb
git commit -m "todo 0.6.0"
git push origin main
```

## 4. Install locally

```bash
brew update
brew upgrade bagadi-alnour/tap/todo
# or: go install github.com/bagadi-alnour/todo-cli/cmd/todo@v0.6.0
```

## 5. Smoke test

```bash
rm -rf .todos   # optional fresh start
todo init
todo add "Ship v0.6.0"
ls .todos/users/
todo list --static
```
