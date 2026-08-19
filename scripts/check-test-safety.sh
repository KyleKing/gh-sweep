#!/usr/bin/env bash
# Static check for test patterns that could reach the real GitHub API.
# Complements the runtime guard in internal/github/transport.go, which panics
# on any real mutating request while `go test` is running.
#
# Run: ./scripts/check-test-safety.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "Checking for unsafe test patterns..."

violations=0

# Test files must not build clients via the raw go-gh constructors; those skip
# the transport seam entirely.
if grep -rn "gh\.RESTClient\|gh\.HTTPClient\|gh\.GQLClient\|NewClientWithToken\|NewClientWithRealAuthAndTransport" --include="*_test.go" .; then
    echo "ERROR: test files construct go-gh clients directly (above)"
    echo "Use github.NewClientWithTransport or github.SetTestTransport instead"
    ((violations++)) || true
fi

# Test files that construct real clients need a fake transport registered in
# the same file (github package tests may reference internals directly).
while IFS= read -r test_file; do
    if grep -q "github\.NewClient(\|github\.NewGQLClient()\|NewClient(context\." "$test_file"; then
        if ! grep -q "SetTestTransport\|NewClientWithTransport\|newTestClient" "$test_file"; then
            echo "WARNING: $test_file constructs a GitHub client but registers no fake transport"
            echo "  Call github.SetTestTransport() or use github.NewClientWithTransport()"
            ((violations++)) || true
        fi
    fi
done < <(find . -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*")

# teatest drives real tea.Cmd goroutines, which construct clients on load;
# those files must register a fake transport.
while IFS= read -r test_file; do
    if grep -q "teatest\." "$test_file"; then
        if ! grep -q "SetTestTransport" "$test_file"; then
            echo "WARNING: $test_file uses teatest without github.SetTestTransport"
            echo "  Component load commands would hit the real GitHub API"
            ((violations++)) || true
        fi
    fi
done < <(find . -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*")

if [ "$violations" -eq 0 ]; then
    echo "OK: no unsafe test patterns detected"
    exit 0
fi

echo ""
echo "Found $violations potential issue(s)"
echo "Note: the runtime guard in internal/github/transport.go still panics on real mutations"
exit 1
