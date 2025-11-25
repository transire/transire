#!/usr/bin/env bash
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

echo "==> gofmt"
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  mapfile -t go_files < <(git ls-files -- '*.go')
else
  mapfile -t go_files < <(find . \
    -path './.git' -prune -o \
    -path './examples/all-handlers-cli/dist' -prune -o \
    -name 'node_modules' -prune -o \
    -name '*.go' -print)
fi
if ((${#go_files[@]})); then
  unformatted=$(gofmt -l "${go_files[@]}" || true)
  if [[ -n "${unformatted}" ]]; then
    echo "gofmt needed for:"
    echo "${unformatted}"
    exit 1
  fi
fi

echo "==> go vet ./..."
go vet ./...

echo "==> go test ./..."
go test ./...

for example in examples/hello examples/all-handlers; do
  if [[ -f "${example}/go.mod" ]]; then
    echo "==> go test ${example}"
    (cd "${example}" && go test ./...)
  fi
done

echo "==> go build ./cmd/transire"
go build ./cmd/transire
