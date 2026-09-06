#!/usr/bin/env bash

# Copyright 2026 MatrixHub Authors
# SPDX-License-Identifier: Apache-2.0

set -u
set -o pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=changelog.sh
source "$SCRIPT_DIR/changelog.sh"

TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/matrixhub-changelog-test.XXXXXX")
PASSED=0
FAILED=0

cleanup() {
  if [ -n "${TEST_ROOT:-}" ] && [ -d "$TEST_ROOT" ]; then
    rm -rf "$TEST_ROOT"
  fi
}
trap cleanup EXIT HUP INT TERM

fail() {
  printf '  %s\n' "$1" >&2
  return 1
}

assert_equal() {
  [ "$1" = "$2" ] || fail "expected <$1>, got <$2>"
}

assert_contains() {
  case "$1" in
    *"$2"*) return 0 ;;
    *) fail "expected output to contain <$2>" ;;
  esac
}

assert_not_contains() {
  case "$1" in
    *"$2"*) fail "expected output not to contain <$2>" ;;
    *) return 0 ;;
  esac
}

new_work_dir() {
  local directory
  directory=$(mktemp -d "$TEST_ROOT/work.XXXXXX") || return 1
  mkdir -p "$directory/entries"
  : > "$directory/errors"
  : > "$directory/warnings"
  : > "$directory/excluded"
  printf '%s\n' "$directory"
}

pull_json() {
  local number=$1
  local body=$2
  local labels=$3
  local author=${4:-contributor}

  jq -cn \
    --argjson number "$number" \
    --arg body "$body" \
    --argjson labels "$labels" \
    --arg author "$author" \
    '{
      number: $number,
      html_url: ("https://github.com/matrixhub-ai/matrixhub/pull/" + ($number | tostring)),
      body: $body,
      labels: ($labels | map({name: .})),
      user: {login: $author}
    }'
}

test_release_note_parsing() {
  local actual
  actual=$(printf 'before\r\n```release-note\r\nFirst line.\r\nSecond line.\r\n```\r\nafter\n' | extract_release_note)
  assert_equal $'First line.\nSecond line.' "$actual" || return 1
  actual=$(printf 'prefix ```release-note Inline note.``` suffix\n' | extract_release_note)
  assert_equal 'Inline note.' "$actual" || return 1
  actual=$(printf '```release-note\n  Padded note.  \n```\n' | extract_release_note)
  assert_equal 'Padded note.' "$actual" || return 1
  assert_equal '' "$(printf '```release-note\nunclosed\n' | extract_release_note)" || return 1
  is_no_release_note ' NONE ' || return 1
  is_no_release_note 'no.' || return 1
  if is_no_release_note '_NONE_'; then
    fail 'underscores are word characters in the ci-bot contract'
  fi
  if is_no_release_note 'No user-visible change'; then
    fail 'descriptive text must not be treated as NONE'
  fi
}

test_version_and_kind_rules() {
  version_is_less v0.9.9 v0.10.0 || return 1
  if version_is_less v0.10.0 v0.9.9; then
    fail 'numeric version ordering is reversed'
  fi
  version_is_less v99999999999999999999.0.0 v100000000000000000000.0.0 || return 1
  assert_equal api-change "$(classify_kind $'kind/feature\nkind/api-change')" || return 1
  assert_equal bug-or-regression "$(classify_kind 'kind/regression')" || return 1
  assert_equal other "$(classify_kind 'kind/cleanup')" || return 1
}

test_previous_official_tag() (
  git_cmd() {
    printf '%s\n' v0.1.0 v0.2.0-rc.1 v0.2.0 v0.10.0 not-a-version
  }

  assert_equal v0.2.0 "$(find_previous_official_tag v0.3.0 HEAD)" || return 1
  assert_equal v0.10.0 "$(find_previous_official_tag v0.11.0 HEAD)" || return 1
  assert_equal '' "$(find_previous_official_tag v0.1.0 HEAD)" || return 1
)

test_pull_validation_and_rendering() {
  local work feature none label_only_none dependabot_body block errors
  work=$(new_work_dir) || return 1
  feature=$(pull_json 21 $'```release-note\nAdded a capability.\nMore detail.\n```' \
    '["kind/feature", "release-note"]' alice) || return 1
  none=$(pull_json 22 $'```release-note\nNONE\n```' \
    '["kind/cleanup", "release-note-none"]' bob) || return 1
  dependabot_body=$'Bumps a dependency from 1.0.0 to 1.0.1.\n\n<details>\n<summary>Release notes</summary>\nUpstream notes.\n</details>'
  label_only_none=$(pull_json 23 "$dependabot_body" \
    '["kind/dependency", "release-note-none"]' 'dependabot[bot]') || return 1

  process_pull "$feature" "$work" || return 1
  process_pull "$none" "$work" || return 1
  process_pull "$label_only_none" "$work" || return 1
  errors=$(cat "$work/errors")
  assert_equal '' "$errors" || return 1
  assert_equal $'22\n23' "$(cat "$work/excluded")" || return 1

  block=$(render_generated_block "$work" matrixhub-ai/matrixhub v0.1.0 v0.2.0) || return 1
  assert_contains "$block" '#### Feature' || return 1
  assert_contains "$block" $'- Added a capability.\n  More detail. ([#21]' || return 1
  assert_contains "$block" '[@alice](https://github.com/alice)' || return 1
  assert_contains "$block" '/compare/v0.1.0...v0.2.0' || return 1
  assert_not_contains "$block" 'NONE' || return 1
}

test_invalid_metadata_is_reported() {
  local work missing_kind conflicting blocked wrong_none missing_release unusable unsupported multiple
  local deprecated_none
  local errors warnings
  work=$(new_work_dir) || return 1
  missing_kind=$(pull_json 30 $'```release-note\nVisible change.\n```' '["release-note"]') || return 1
  conflicting=$(pull_json 31 $'```release-note\nNONE\n```' \
    '["kind/cleanup", "release-note", "release-note-none"]') || return 1
  blocked=$(pull_json 32 $'```release-note\nFix.\n```' \
    '["kind/bug", "release-note", "do-not-merge/needs-kind", "do-not-merge/needs-release-note"]') || return 1
  wrong_none=$(pull_json 33 $'```release-note\nVisible change.\n```' \
    '["kind/feature", "release-note-none"]') || return 1
  missing_release=$(pull_json 34 $'```release-note\nFix.\n```' '["kind/bug"]') || return 1
  unusable=$(pull_json 35 $'```release-note\nNONE\n```' \
    '["kind/bug", "release-note"]') || return 1
  unsupported=$(pull_json 36 $'```release-note\nChange.\n```' \
    '["kind/unknown", "release-note"]') || return 1
  multiple=$(pull_json 37 $'```release-note\nAPI change.\n```' \
    '["kind/feature", "kind/api-change", "release-note"]') || return 1
  deprecated_none=$(pull_json 38 '' \
    '["kind/deprecation", "release-note-none"]') || return 1

  process_pull "$missing_kind" "$work" || return 1
  process_pull "$conflicting" "$work" || return 1
  process_pull "$blocked" "$work" || return 1
  process_pull "$wrong_none" "$work" || return 1
  process_pull "$missing_release" "$work" || return 1
  process_pull "$unusable" "$work" || return 1
  process_pull "$unsupported" "$work" || return 1
  process_pull "$multiple" "$work" || return 1
  process_pull "$deprecated_none" "$work" || return 1
  errors=$(cat "$work/errors")
  warnings=$(cat "$work/warnings")
  assert_contains "$errors" '#30 has no kind/* label' || return 1
  assert_contains "$errors" '#31 has both release-note and release-note-none labels' || return 1
  assert_contains "$errors" '#32 is still labeled do-not-merge/needs-kind' || return 1
  assert_contains "$errors" '#32 is still labeled do-not-merge/needs-release-note' || return 1
  assert_contains "$errors" '#33 has release-note-none but its release-note block contains content' || return 1
  assert_contains "$errors" '#34 has neither release-note nor release-note-none' || return 1
  assert_contains "$errors" '#35 has release-note but no usable release-note content' || return 1
  assert_contains "$errors" '#36 has unsupported kind labels: kind/unknown' || return 1
  assert_contains "$errors" '#38 has kind/deprecation and release-note-none' || return 1
  assert_contains "$warnings" '#37 has multiple kind labels (kind/feature, kind/api-change); classified as api-change' || return 1
}

test_release_note_is_not_executed() {
  local work sentinel body pull output
  work=$(new_work_dir) || return 1
  sentinel="$work/should-not-exist"
  body="\`\`\`release-note
\$(touch $sentinel)
\`\`\`"
  pull=$(pull_json 40 "$body" '["kind/feature", "release-note"]') || return 1
  process_pull "$pull" "$work" || return 1
  [ ! -e "$sentinel" ] || fail 'release-note content was executed' || return 1
  output=$(cat "$work/entries/feature/000000000040.md")
  assert_contains "$output" "\$(touch $sentinel)" || return 1
}

test_pull_discovery_filters_and_deduplicates() (
  local work pulls warnings
  work=$(new_work_dir) || return 1

  git_cmd() {
    case "$1" in
      rev-list) printf '%s\n' aaaaaaaaaaaa1111 bbbbbbbbbbbb2222 cccccccccccc3333 ;;
      show) printf 'direct commit\n' ;;
      *) return 90 ;;
    esac
  }
  github_api() {
    case "$1" in
      *aaaaaaaaaaaa1111*)
        printf '%s\n' '[{"number": 5, "merged_at": "2026-01-01", "base": {"ref": "main"}}]'
        ;;
      *bbbbbbbbbbbb2222*)
        printf '%s\n' '[{"number": 5, "merged_at": "2026-01-01", "base": {"ref": "main"}}, {"number": 6, "merged_at": "2026-01-01", "base": {"ref": "release"}}]'
        ;;
      *cccccccccccc3333*) printf '%s\n' '[]' ;;
      *) return 91 ;;
    esac
  }

  collect_pull_numbers v0.1.0 HEAD main matrixhub-ai/matrixhub "$work" || return 1
  pulls=$(cat "$work/pull_numbers.sorted")
  warnings=$(cat "$work/warnings")
  assert_equal 5 "$pulls" || return 1
  assert_contains "$warnings" 'no merged PR found for cccccccccccc direct commit' || return 1
)

test_commit_message_pull_discovery() (
  local work pulls warnings commit_sha
  work=$(new_work_dir) || return 1
  commit_sha=aaaaaaaaaaaa1111

  assert_equal 123 "$(printf 'Merge pull request #123 from topic\n\nTitle\n' | parse_pull_number_from_commit_message)" || return 1
  assert_equal 34 "$(printf 'docs: mention (#12) and finish (#34)\n' | parse_pull_number_from_commit_message)" || return 1
  assert_equal '' "$(printf 'direct commit\n' | parse_pull_number_from_commit_message)" || return 1

  git_cmd() {
    case "$1" in
      rev-list) printf '%s\n' "$commit_sha" ;;
      show) printf 'Merge pull request #12 from topic\n\nFeature\n' ;;
      *) return 90 ;;
    esac
  }
  github_api() {
    case "$1" in
      repos/*/pulls/12)
        [ "${2:-}" = --include ] || return 93
        printf 'HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n'
        printf '{"number":12,"merged_at":"2026-01-01","base":{"ref":"main"},"merge_commit_sha":"%s"}\n' "$commit_sha"
        ;;
      repos/*/commits/*)
        fail 'associated PR lookup should not run after a verified commit-message match'
        return 91
        ;;
      *) return 92 ;;
    esac
  }

  collect_pull_numbers v0.1.0 HEAD main matrixhub-ai/matrixhub "$work" || return 1
  pulls=$(cat "$work/pull_numbers.sorted")
  warnings=$(cat "$work/warnings")
  assert_equal 12 "$pulls" || return 1
  assert_equal '' "$warnings" || return 1
)

test_commit_message_pr_lookup_errors() (
  local work error mock_api_status=404
  work=$(new_work_dir) || return 1
  error="$work/error"

  git_cmd() {
    case "$1" in
      rev-list) printf '%s\n' aaaaaaaaaaaa1111 ;;
      show) printf 'fix from a referenced pull request (#404)\n' ;;
      *) return 90 ;;
    esac
  }
  github_api() {
    case "$1" in
      repos/*/pulls/404)
        [ "${2:-}" = --include ] || return 93
        printf 'HTTP/2.0 %s Error\r\nContent-Type: application/json\r\n\r\n' "$mock_api_status"
        printf '{"status":"%s"}\n' "$mock_api_status"
        if [ "$mock_api_status" = 404 ]; then
          printf 'opaque client error\n' >&2
        else
          printf 'gh: Not Found (HTTP 404)\n' >&2
        fi
        return 1
        ;;
      repos/*/commits/*)
        printf '%s\n' '[{"number":45,"merged_at":"2026-01-01","base":{"ref":"main"}}]'
        ;;
      *) return 92 ;;
    esac
  }

  collect_pull_numbers v0.1.0 HEAD main matrixhub-ai/matrixhub "$work" || return 1
  assert_equal 45 "$(cat "$work/pull_numbers.sorted")" || return 1

  work=$(new_work_dir) || return 1
  error="$work/error"
  mock_api_status=500
  if collect_pull_numbers v0.1.0 HEAD main matrixhub-ai/matrixhub "$work" 2> "$error"; then
    fail 'a non-404 PR lookup error should stop collection'
    return 1
  fi
  assert_contains "$(cat "$error")" 'gh: Not Found (HTTP 404)' || return 1
  assert_contains "$(cat "$error")" 'failed to read pull request #404' || return 1
)

test_prepare_workflow_uses_snapshot_lease() {
  local workflow
  workflow=$(cat "$SCRIPT_DIR/../.github/workflows/prepare-release-notes.yml") || return 1

  assert_contains "$workflow" 'echo "expected_remote_sha=${expected_remote_sha}" >> "${GITHUB_OUTPUT}"' || return 1
  assert_contains "$workflow" 'EXPECTED_REMOTE_SHA: ${{ steps.draft.outputs.expected_remote_sha }}' || return 1
  assert_contains "$workflow" '--force-with-lease="refs/heads/${RELEASE_BRANCH}:${EXPECTED_REMOTE_SHA}"' || return 1
  assert_not_contains "$workflow" 'git push --force-with-lease --set-upstream' || return 1
}

test_changelog_create_and_prepend() {
  local work generated changelog blank_changelog content newest older
  work=$(new_work_dir) || return 1
  generated="$work/generated.md"
  changelog="$work/CHANGELOG.md"
  printf '%s\nnew generated content\n%s\n' "$GENERATED_START" "$GENERATED_END" > "$generated"

  upsert_changelog "$changelog" "$changelog" v0.2.0 "$generated" "$work" || return 1
  content=$(cat "$changelog")
  assert_contains "$content" '# MatrixHub v0.2.x release notes' || return 1
  assert_contains "$content" '## v0.2.0' || return 1

  blank_changelog="$work/BLANK.md"
  printf '\n  \n' > "$blank_changelog"
  upsert_changelog "$blank_changelog" "$blank_changelog" v0.2.0 "$generated" "$work" || return 1
  assert_contains "$(cat "$blank_changelog")" '# MatrixHub v0.2.x release notes' || return 1

  upsert_changelog "$changelog" "$changelog" v0.2.1 "$generated" "$work" || return 1
  newest=$(awk '$0 == "## v0.2.1" { print NR; exit }' "$changelog")
  older=$(awk '$0 == "## v0.2.0" { print NR; exit }' "$changelog")
  [ "$newest" -lt "$older" ] || fail 'new version was not inserted before the old version'
}

test_changelog_upsert_is_safe_and_idempotent() {
  local work generated changelog before
  work=$(new_work_dir) || return 1
  generated="$work/generated.md"
  changelog="$work/CHANGELOG.md"
  printf '%s\nnew generated content\n%s\n' "$GENERATED_START" "$GENERATED_END" > "$generated"
  printf '%s\n' \
    '# MatrixHub v0.2.x release notes' \
    '' \
    '## v0.2.0' \
    '' \
    '### Overview' \
    '' \
    'Keep this overview.' \
    '' \
    "$GENERATED_START" \
    'old generated content' \
    "$GENERATED_END" \
    '' \
    '### Known Issues' \
    '' \
    'Keep this issue.' > "$changelog"

  upsert_changelog "$changelog" "$changelog" v0.2.0 "$generated" "$work" || return 1
  before=$(cat "$changelog")
  assert_contains "$before" 'Keep this overview.' || return 1
  assert_contains "$before" 'Keep this issue.' || return 1
  assert_contains "$before" 'new generated content' || return 1
  assert_not_contains "$before" 'old generated content' || return 1

  upsert_changelog "$changelog" "$changelog" v0.2.0 "$generated" "$work" || return 1
  assert_equal "$before" "$(cat "$changelog")" || return 1
}

test_changelog_refuses_unmarked_section() {
  local work generated changelog original before
  work=$(new_work_dir) || return 1
  generated="$work/generated.md"
  changelog="$work/CHANGELOG.md"
  printf '%s\nnew block\n%s\n' "$GENERATED_START" "$GENERATED_END" > "$generated"
  original=$'## v0.2.0\n\nHand-written notes.\n'
  printf '%s' "$original" > "$changelog"
  before=$(cksum < "$changelog")

  if upsert_changelog "$changelog" "$changelog" v0.2.0 "$generated" "$work" 2>/dev/null; then
    fail 'unmarked section should be rejected'
    return 1
  fi
  assert_equal "$before" "$(cksum < "$changelog")" || return 1
}

test_changelog_rejects_non_regular_target() {
  local work generated changelog
  work=$(new_work_dir) || return 1
  generated="$work/generated.md"
  changelog="$work/CHANGELOG.md"
  printf '%s\nnew block\n%s\n' "$GENERATED_START" "$GENERATED_END" > "$generated"
  mkdir -p "$changelog"

  if upsert_changelog "$changelog" "$changelog" v0.2.0 "$generated" "$work" 2>/dev/null; then
    fail 'a directory changelog target should be rejected'
    return 1
  fi
  [ ! -e "$changelog/changelog.updated" ] || fail 'content was written inside the target directory'
}

test_full_cli_contract() {
  local case_dir repo_dir bin_dir associated pull stdout stderr sha output
  local github_output summary write_stdout invalid_stderr hostile_tmp sentinel before
  case_dir=$(mktemp -d "$TEST_ROOT/cli.XXXXXX") || return 1
  repo_dir="$case_dir/repo"
  bin_dir="$case_dir/bin"
  mkdir -p "$repo_dir" "$bin_dir"

  git -C "$repo_dir" init -q || return 1
  git -C "$repo_dir" config user.name test || return 1
  git -C "$repo_dir" config user.email test@example.com || return 1
  printf 'seed\n' > "$repo_dir/content"
  git -C "$repo_dir" add content || return 1
  git -C "$repo_dir" commit -q -m seed || return 1
  printf 'change\n' >> "$repo_dir/content"
  git -C "$repo_dir" commit -qam 'feat: change' || return 1
  sha=$(git -C "$repo_dir" rev-parse HEAD) || return 1

  associated="$case_dir/associated.json"
  pull="$case_dir/pull.json"
  printf '[{"number":42,"merged_at":"2026-01-01","base":{"ref":"main"}}]\n' > "$associated"
  pull_json 42 $'```release-note\nAdded from the CLI.\n```' \
    '["kind/feature", "release-note"]' cli-author > "$pull" || return 1

  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'endpoint=${!#}' \
    'case "$endpoint" in' \
    '  repos/*/commits/*/pulls*) cat "$MOCK_ASSOCIATED" ;;' \
    '  repos/*/pulls/42) cat "$MOCK_PULL" ;;' \
    '  *) echo "unexpected endpoint: $endpoint" >&2; exit 97 ;;' \
    'esac' > "$bin_dir/gh"
  chmod +x "$bin_dir/gh"

  stdout="$case_dir/stdout"
  stderr="$case_dir/stderr"
  github_output="$case_dir/github-output"
  summary="$case_dir/summary"
  if ! (
    cd "$repo_dir" || exit 1
    PATH="$bin_dir:$PATH" \
      GH_TOKEN='secret-test-token' \
      GITHUB_OUTPUT="$github_output" \
      GITHUB_STEP_SUMMARY="$summary" \
      MOCK_ASSOCIATED="$associated" \
      MOCK_PULL="$pull" \
      /bin/bash "$SCRIPT_DIR/changelog.sh" \
        --version v0.2.0 \
        --repo matrixhub-ai/matrixhub \
        --end-ref HEAD \
        --base-branch main \
        --dry-run
  ) > "$stdout" 2> "$stderr"; then
    cat "$stderr" >&2
    fail 'full CLI dry-run failed'
    return 1
  fi

  output=$(cat "$stdout")
  assert_contains "$output" 'Collecting v0.2.0 release notes from HEAD' || return 1
  assert_contains "$output" 'Added from the CLI.' || return 1
  assert_contains "$output" '[@cli-author](https://github.com/cli-author)' || return 1
  assert_not_contains "$output$(cat "$stderr")$(cat "$github_output")$(cat "$summary")" 'secret-test-token' || return 1
  assert_contains "$(cat "$github_output")" 'output_path=CHANGELOG/CHANGELOG-0.2.md' || return 1
  assert_contains "$(cat "$github_output")" 'start_ref=repository-root' || return 1
  assert_contains "$(cat "$github_output")" 'included_count=1' || return 1
  assert_contains "$(cat "$github_output")" 'excluded_count=0' || return 1
  assert_contains "$(cat "$github_output")" 'pull_count=1' || return 1
  assert_contains "$(cat "$summary")" 'Range: `repository-root..HEAD`' || return 1
  assert_contains "$(cat "$summary")" 'Commits without an associated merged PR: 0' || return 1
  [ ! -e "$repo_dir/CHANGELOG/CHANGELOG-0.2.md" ] || fail 'dry-run wrote CHANGELOG' || return 1
  [ -n "$sha" ] || fail 'test commit was not created' || return 1

  write_stdout="$case_dir/write-stdout"
  if ! (
    cd "$repo_dir" || exit 1
    PATH="$bin_dir:$PATH" \
      GH_TOKEN='secret-test-token' \
      MOCK_ASSOCIATED="$associated" \
      MOCK_PULL="$pull" \
      /bin/bash "$SCRIPT_DIR/changelog.sh" \
        --version v0.2.0 \
        --repo matrixhub-ai/matrixhub \
        --end-ref HEAD \
        --base-branch main
  ) > "$write_stdout" 2> "$stderr"; then
    cat "$stderr" >&2
    fail 'full CLI write failed'
    return 1
  fi
  assert_contains "$(cat "$write_stdout")" 'Updated CHANGELOG/CHANGELOG-0.2.md' || return 1
  assert_contains "$(cat "$repo_dir/CHANGELOG/CHANGELOG-0.2.md")" 'Added from the CLI.' || return 1
  before=$(cksum < "$repo_dir/CHANGELOG/CHANGELOG-0.2.md")

  pull_json 42 $'```release-note\nMissing Kind.\n```' '["release-note"]' cli-author > "$pull" || return 1
  hostile_tmp="$case_dir/tmp'; touch trap-ran; #"
  sentinel="$repo_dir/trap-ran"
  invalid_stderr="$case_dir/invalid-stderr"
  mkdir -p "$hostile_tmp"
  if (
    cd "$repo_dir" || exit 1
    PATH="$bin_dir:$PATH" \
      GH_TOKEN='secret-test-token' \
      TMPDIR="$hostile_tmp" \
      MOCK_ASSOCIATED="$associated" \
      MOCK_PULL="$pull" \
      /bin/bash "$SCRIPT_DIR/changelog.sh" \
        --version v0.2.0 \
        --repo matrixhub-ai/matrixhub \
        --end-ref HEAD \
        --base-branch main
  ) > /dev/null 2> "$invalid_stderr"; then
    fail 'invalid PR metadata should fail the full CLI'
    return 1
  fi
  assert_contains "$(cat "$invalid_stderr")" '#42 has no kind/* label' || return 1
  assert_equal "$before" "$(cksum < "$repo_dir/CHANGELOG/CHANGELOG-0.2.md")" || return 1
  [ ! -e "$sentinel" ] || fail 'cleanup trap executed TMPDIR content' || return 1
  set -- "$hostile_tmp"/matrixhub-changelog.*
  [ ! -e "$1" ] || fail 'failed CLI left its work directory behind'
}

run_test() {
  local name=$1
  shift
  if "$@"; then
    PASSED=$((PASSED + 1))
    printf 'ok %d - %s\n' "$PASSED" "$name"
  else
    FAILED=$((FAILED + 1))
    printf 'not ok - %s\n' "$name"
  fi
}

run_test 'release-note parsing' test_release_note_parsing
run_test 'version and Kind rules' test_version_and_kind_rules
run_test 'previous official tag' test_previous_official_tag
run_test 'PR validation and rendering' test_pull_validation_and_rendering
run_test 'invalid metadata errors' test_invalid_metadata_is_reported
run_test 'release-note content is data' test_release_note_is_not_executed
run_test 'PR discovery filters and deduplicates' test_pull_discovery_filters_and_deduplicates
run_test 'commit-message PR discovery' test_commit_message_pull_discovery
run_test 'commit-message PR lookup errors' test_commit_message_pr_lookup_errors
run_test 'release workflow snapshot lease' test_prepare_workflow_uses_snapshot_lease
run_test 'CHANGELOG creation and prepend' test_changelog_create_and_prepend
run_test 'CHANGELOG upsert and idempotence' test_changelog_upsert_is_safe_and_idempotent
run_test 'unmarked section refusal' test_changelog_refuses_unmarked_section
run_test 'non-regular target refusal' test_changelog_rejects_non_regular_target
run_test 'full CLI contract' test_full_cli_contract

printf '1..%d\n' "$((PASSED + FAILED))"
[ "$FAILED" -eq 0 ]
