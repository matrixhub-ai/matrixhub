#!/usr/bin/env bash

# Copyright 2026 MatrixHub Authors
# SPDX-License-Identifier: Apache-2.0

GENERATED_START='<!-- BEGIN GENERATED RELEASE NOTES -->'
GENERATED_END='<!-- END GENERATED RELEASE NOTES -->'
CHANGELOG_WORK_DIR=''

usage() {
  cat <<'EOF'
Usage:
  scripts/changelog.sh --version vX.Y.Z [options]

Options:
  --repo OWNER/REPO       GitHub repository (defaults to GITHUB_REPOSITORY)
  --start-ref REF         Previous official release (auto-detected when omitted)
  --end-ref REF           Candidate commit or branch (default: HEAD)
  --base-branch BRANCH    PR base branch used to disambiguate associated PRs
  --output PATH           CHANGELOG file to update (default: CHANGELOG/CHANGELOG-X.Y.md)
  --dry-run               Print the generated version section without writing a file
  --help                  Show this help
EOF
}

git_cmd() {
  git "$@"
}

github_api() {
  local endpoint=$1
  shift

  gh api \
    -H 'Accept: application/vnd.github+json' \
    -H 'X-GitHub-Api-Version: 2022-11-28' \
    "$@" \
    "$endpoint"
}

http_status_code() {
  awk 'NR == 1 && /^HTTP\// { print $2; exit }'
}

http_response_body() {
  awk '
    NR == 1 && /^HTTP\// { headers = 1; next }
    headers {
      sub(/\r$/, "")
      if ($0 == "") headers = 0
      next
    }
    { print }
  '
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    return 1
  }
}

append_line() {
  printf '%s\n' "$2" >> "$1"
}

has_line() {
  local lines=$1
  local expected=$2
  local line

  while IFS= read -r line; do
    [ "$line" = "$expected" ] && return 0
  done <<EOF
$lines
EOF
  return 1
}

trim_blank_lines() {
  awk '
    { lines[NR] = $0 }
    END {
      first = 1
      while (first <= NR && lines[first] ~ /^[[:space:]]*$/) first++
      last = NR
      while (last >= first && lines[last] ~ /^[[:space:]]*$/) last--
      if (first <= last) {
        sub(/^[[:space:]]+/, "", lines[first])
        sub(/[[:space:]]+$/, "", lines[last])
      }
      for (i = first; i <= last; i++) print lines[i]
    }
  '
}

extract_release_note() {
  awk '
    function remember(value) {
      content[++count] = value
    }
    {
      line = $0
      sub(/\r$/, "", line)
      lower = tolower(line)
      if (!found && match(lower, /```release-note[[:blank:]]*/)) {
        found = 1
        rest = substr(line, RSTART + RLENGTH)
        closing = index(rest, "```")
        if (closing) {
          remember(substr(rest, 1, closing - 1))
          closed = 1
          exit
        }
        remember(rest)
        next
      }
      if (found) {
        closing = index(line, "```")
        if (closing) {
          remember(substr(line, 1, closing - 1))
          closed = 1
          exit
        }
        remember(line)
      }
    }
    END {
      if (closed) {
        for (i = 1; i <= count; i++) print content[i]
      }
    }
  ' | trim_blank_lines
}

is_no_release_note() {
  local upper
  upper=$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]')
  [[ $upper =~ ^[^A-Z0-9_]*(NONE|NO)[^A-Z0-9_]*$ ]]
}

version_is_less() {
  local left=${1#v}
  local right=${2#v}
  local -a left_parts right_parts
  local index left_part right_part left_digit right_digit

  IFS=. read -r -a left_parts <<EOF
$left
EOF
  IFS=. read -r -a right_parts <<EOF
$right
EOF

  for index in 0 1 2; do
    left_part=${left_parts[$index]}
    right_part=${right_parts[$index]}
    while [ "${#left_part}" -gt 1 ] && [ "${left_part#0}" != "$left_part" ]; do
      left_part=${left_part#0}
    done
    while [ "${#right_part}" -gt 1 ] && [ "${right_part#0}" != "$right_part" ]; do
      right_part=${right_part#0}
    done
    [ "${#left_part}" -lt "${#right_part}" ] && return 0
    [ "${#left_part}" -gt "${#right_part}" ] && return 1
    while [ -n "$left_part" ]; do
      left_digit=${left_part%"${left_part#?}"}
      right_digit=${right_part%"${right_part#?}"}
      [ "$left_digit" -lt "$right_digit" ] && return 0
      [ "$left_digit" -gt "$right_digit" ] && return 1
      left_part=${left_part#?}
      right_part=${right_part#?}
    done
  done
  return 1
}

find_previous_official_tag() {
  local version=$1
  local end_ref=$2
  local tag tags
  local best=''

  tags=$(git_cmd tag --merged "$end_ref" --list) || return 1
  while IFS= read -r tag; do
    [[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue
    version_is_less "$tag" "$version" || continue
    if [ -z "$best" ] || version_is_less "$best" "$tag"; then
      best=$tag
    fi
  done <<EOF
$tags
EOF
  printf '%s\n' "$best"
}

parse_pull_number_from_commit_message() {
  awk '
    /^Merge pull request #[0-9]+/ {
      value = $0
      sub(/^Merge pull request #/, "", value)
      sub(/[^0-9].*$/, "", value)
      print value
      found = 1
      exit
    }
    {
      remaining = $0
      while (match(remaining, /\(#[0-9]+\)/)) {
        value = substr(remaining, RSTART + 2, RLENGTH - 3)
        candidate = value
        remaining = substr(remaining, RSTART + RLENGTH)
      }
    }
    END {
      if (!found && candidate != "") print candidate
    }
  '
}

classify_kind() {
  local labels=$1
  local label

  has_line "$labels" 'kind/deprecation' && { printf 'deprecation\n'; return; }
  has_line "$labels" 'kind/api-change' && { printf 'api-change\n'; return; }
  has_line "$labels" 'kind/feature' && { printf 'feature\n'; return; }
  if has_line "$labels" 'kind/bug' || has_line "$labels" 'kind/regression'; then
    printf 'bug-or-regression\n'
    return
  fi
  has_line "$labels" 'kind/documentation' && { printf 'documentation\n'; return; }
  for label in \
    kind/cleanup \
    kind/dependency \
    kind/design \
    kind/failing-test \
    kind/flake \
    kind/support; do
    has_line "$labels" "$label" && { printf 'other\n'; return; }
  done
  return 1
}

join_lines() {
  awk 'NF { if (seen++) printf ", "; printf "%s", $0 } END { print "" }'
}

process_pull() {
  local pull_json=$1
  local work_dir=$2
  local number url author body labels note
  local kind_labels='' kind_count=0 label
  local has_release_note=false has_release_note_none=false category entry_file

  number=$(printf '%s' "$pull_json" | jq -er '.number')
  url=$(printf '%s' "$pull_json" | jq -er '.html_url')
  author=$(printf '%s' "$pull_json" | jq -r '.user.login // "ghost"')
  body=$(printf '%s' "$pull_json" | jq -r '.body // ""')
  labels=$(printf '%s' "$pull_json" | jq -r '.labels[]? | if type == "string" then . else .name end')
  note=$(printf '%s\n' "$body" | extract_release_note)

  while IFS= read -r label; do
    case "$label" in
      kind/*)
        if [ -z "$kind_labels" ]; then
          kind_labels=$label
        else
          kind_labels="$kind_labels
$label"
        fi
        kind_count=$((kind_count + 1))
        ;;
    esac
  done <<EOF
$labels
EOF

  [ "$kind_count" -gt 0 ] || append_line "$work_dir/errors" "#$number has no kind/* label"
  has_line "$labels" 'do-not-merge/needs-kind' && \
    append_line "$work_dir/errors" "#$number is still labeled do-not-merge/needs-kind"
  has_line "$labels" 'do-not-merge/needs-release-note' && \
    append_line "$work_dir/errors" "#$number is still labeled do-not-merge/needs-release-note"
  has_line "$labels" 'release-note' && has_release_note=true
  has_line "$labels" 'release-note-none' && has_release_note_none=true

  if [ "$has_release_note" = true ] && [ "$has_release_note_none" = true ]; then
    append_line "$work_dir/errors" "#$number has both release-note and release-note-none labels"
    return
  fi

  if [ "$has_release_note_none" = true ]; then
    if [ -n "$note" ] && ! is_no_release_note "$note"; then
      append_line "$work_dir/errors" \
        "#$number has release-note-none but its release-note block contains content"
      return
    fi
    if has_line "$kind_labels" 'kind/deprecation'; then
      append_line "$work_dir/errors" "#$number has kind/deprecation and release-note-none"
      return
    fi
    append_line "$work_dir/excluded" "$number"
    return
  fi

  if [ "$has_release_note" = false ]; then
    append_line "$work_dir/errors" "#$number has neither release-note nor release-note-none"
    return
  fi

  if [ -z "$note" ] || is_no_release_note "$note"; then
    append_line "$work_dir/errors" "#$number has release-note but no usable release-note content"
    return
  fi

  if ! category=$(classify_kind "$kind_labels"); then
    append_line "$work_dir/errors" \
      "#$number has unsupported kind labels: $(printf '%s\n' "$kind_labels" | join_lines)"
    return
  fi

  if [ "$kind_count" -gt 1 ]; then
    append_line "$work_dir/warnings" \
      "#$number has multiple kind labels ($(printf '%s\n' "$kind_labels" | join_lines)); classified as $category"
  fi

  mkdir -p "$work_dir/entries/$category"
  entry_file=$(printf '%s/entries/%s/%012d.md' "$work_dir" "$category" "$number")
  {
    printf -- '- '
    printf '%s\n' "$note" | awk 'NR == 1 { printf "%s", $0; next } { printf "\n  %s", $0 }'
    printf ' ([#%s](%s), [@%s](https://github.com/%s))\n\n' \
      "$number" "$url" "$author" "$author"
  } > "$entry_file"
}

collect_pull_numbers() {
  local start_ref=$1
  local end_ref=$2
  local base_branch=$3
  local repo=$4
  local work_dir=$5
  local range=$end_ref sha pulls numbers number subject message message_pull_number
  local direct_pull api_error api_response api_status found

  [ -z "$start_ref" ] || range="$start_ref..$end_ref"
  git_cmd rev-list --reverse "$range" > "$work_dir/commits"
  : > "$work_dir/pull_numbers"

  while IFS= read -r sha; do
    [ -n "$sha" ] || continue
    message=$(git_cmd show -s --format=%s%n%b "$sha")
    subject=${message%%$'\n'*}
    message_pull_number=$(printf '%s\n' "$message" | parse_pull_number_from_commit_message)
    found=false

    if [ -n "$message_pull_number" ]; then
      api_error="$work_dir/github-api-error"
      : > "$api_error"
      if api_response=$(github_api "repos/$repo/pulls/$message_pull_number" \
        --include 2> "$api_error"); then
        direct_pull=$(printf '%s\n' "$api_response" | http_response_body)
        if printf '%s' "$direct_pull" | jq -e \
          --arg base "$base_branch" \
          --arg sha "$sha" \
          '.merged_at != null
            and ($base == "" or .base.ref == $base)
            and .merge_commit_sha == $sha' >/dev/null; then
          append_line "$work_dir/pull_numbers" "$message_pull_number"
          found=true
        fi
      else
        api_status=$(printf '%s\n' "$api_response" | http_status_code)
        if [ "$api_status" != 404 ]; then
          cat "$api_error" >&2
          echo "error: failed to read pull request #$message_pull_number" >&2
          return 1
        fi
      fi
    fi

    if [ "$found" = false ]; then
      if ! pulls=$(github_api "repos/$repo/commits/$sha/pulls?per_page=100"); then
        echo "error: failed to find pull requests associated with commit $sha" >&2
        return 1
      fi
      numbers=$(printf '%s' "$pulls" | jq -r --arg base "$base_branch" '
        .[]
        | select(.merged_at != null)
        | select($base == "" or .base.ref == $base)
        | .number
      ')
      if [ -z "$numbers" ]; then
        append_line "$work_dir/warnings" \
          "no merged PR found for $(printf '%.12s' "$sha") $subject"
        continue
      fi
      while IFS= read -r number; do
        [ -n "$number" ] && append_line "$work_dir/pull_numbers" "$number"
      done <<EOF
$numbers
EOF
    fi
  done < "$work_dir/commits"

  sort -n -u "$work_dir/pull_numbers" > "$work_dir/pull_numbers.sorted"
}

render_generated_block() {
  local work_dir=$1
  local repo=$2
  local start_ref=$3
  local version=$4
  local category title directory file
  local found=false compare_url

  printf '%s\n' "$GENERATED_START"
  printf '%s\n' '<!-- Generated by scripts/changelog.sh. Edit PR release-note blocks'
  printf '%s\n' '     and rerun the workflow to change this block. Add hand-written overview,'
  printf '%s\n\n' '     upgrade notes, and known issues outside this block. -->'
  printf '%s\n\n' '### Changes by Kind'

  while IFS='|' read -r category title; do
    directory="$work_dir/entries/$category"
    [ -d "$directory" ] || continue
    set -- "$directory"/*.md
    [ -e "$1" ] || continue
    found=true
    printf '#### %s\n\n' "$title"
    for file in "$directory"/*.md; do
      cat "$file"
    done
  done <<'EOF'
deprecation|Deprecation
api-change|API Change
feature|Feature
bug-or-regression|Bug or Regression
documentation|Documentation
other|Other
EOF

  if [ "$found" = false ]; then
    printf '%s\n\n' 'No user-facing, API-facing, or operator-facing changes were reported for this release.'
  fi

  if [ -n "$start_ref" ]; then
    compare_url="https://github.com/$repo/compare/$start_ref...$version"
  else
    compare_url="https://github.com/$repo/commits/$version"
  fi
  printf '[Full Changelog](%s)\n\n%s\n' "$compare_url" "$GENERATED_END"
}

upsert_changelog() {
  local existing_file=$1
  local output_file=$2
  local version=$3
  local generated_file=$4
  local work_dir=$5
  local version_number=${version#v}
  local major=${version_number%%.*}
  local remainder=${version_number#*.}
  local minor=${remainder%%.*}
  local source=/dev/null
  local temporary="$work_dir/changelog.updated"

  if [ -e "$existing_file" ] || [ -L "$existing_file" ]; then
    if [ ! -f "$existing_file" ] || [ -L "$existing_file" ]; then
      echo "error: changelog target is not a regular file: $existing_file" >&2
      return 1
    fi
    source=$existing_file
  fi
  if [ "$output_file" != "$existing_file" ] && \
    { [ -L "$output_file" ] || { [ -e "$output_file" ] && [ ! -f "$output_file" ]; }; }; then
    echo "error: changelog target is not a regular file: $output_file" >&2
    return 1
  fi
  if ! awk \
    -v version="$version" \
    -v major="$major" \
    -v minor="$minor" \
    -v block_file="$generated_file" \
    -v generated_start="$GENERATED_START" \
    -v generated_end="$GENERATED_END" '
      BEGIN {
        while ((getline line < block_file) > 0) block[++block_count] = line
        close(block_file)
      }
      {
        sub(/\r$/, "")
        lines[NR] = $0
      }
      function emit_block(  i) {
        for (i = 1; i <= block_count; i++) print block[i]
      }
      function emit_section() {
        print "## " version
        print ""
        emit_block()
        print ""
        print "---"
        print ""
      }
      END {
        count = NR
        heading = 0
        first_heading = 0
        has_content = 0
        for (i = 1; i <= count; i++) {
          value = lines[i]
          sub(/[[:space:]]+$/, "", value)
          if (value != "") has_content = 1
          if (!first_heading && value ~ /^## v[0-9]+\.[0-9]+\.[0-9]+$/) first_heading = i
          if (value == "## " version) heading = i
        }

        if (heading) {
          section_end = count + 1
          for (i = heading + 1; i <= count; i++) {
            value = lines[i]
            sub(/[[:space:]]+$/, "", value)
            if (value ~ /^## v[0-9]+\.[0-9]+\.[0-9]+$/) {
              section_end = i
              break
            }
          }
          block_start = 0
          block_end = 0
          for (i = heading + 1; i < section_end; i++) {
            if (!block_start && lines[i] == generated_start) block_start = i
            if (block_start && lines[i] == generated_end) {
              block_end = i
              break
            }
          }
          if (!block_start || !block_end) {
            print "error: the " version " section exists without a complete generated block; refusing to overwrite hand-written content" > "/dev/stderr"
            exit 2
          }
          for (i = 1; i < block_start; i++) print lines[i]
          emit_block()
          for (i = block_end + 1; i <= count; i++) print lines[i]
          exit
        }

        if (!count || !has_content) {
          print "# MatrixHub v" major "." minor ".x release notes"
          print ""
          emit_section()
          exit
        }

        if (first_heading) {
          for (i = 1; i < first_heading; i++) print lines[i]
          emit_section()
          for (i = first_heading; i <= count; i++) print lines[i]
          exit
        }

        for (i = 1; i <= count; i++) print lines[i]
        if (lines[count] != "") print ""
        emit_section()
      }
    ' "$source" > "$temporary"; then
    return 1
  fi

  mkdir -p "$(dirname "$output_file")"
  mv "$temporary" "$output_file"
}

line_count() {
  if [ -s "$1" ]; then
    wc -l < "$1" | tr -d ' '
  else
    printf '0\n'
  fi
}

file_count() {
  if [ -d "$1" ]; then
    find "$1" -type f | wc -l | tr -d ' '
  else
    printf '0\n'
  fi
}

append_github_output() {
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"
  fi
}

cleanup_work_dir() {
  local directory=${1:-}
  if [ -n "$directory" ] && [ -d "$directory" ]; then
    rm -rf "$directory"
  fi
}

main() {
  local version=''
  local repo=${GITHUB_REPOSITORY:-}
  local start_ref=''
  local end_ref=HEAD
  local base_branch=${GITHUB_BASE_REF:-${GITHUB_REF_NAME:-}}
  local output=''
  local dry_run=false
  local work_dir token range number pull_json
  local included_count excluded_count pull_count commit_count warning

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --dry-run)
        dry_run=true
        shift
        ;;
      --help)
        usage
        return
        ;;
      --version|--repo|--start-ref|--end-ref|--base-branch|--output)
        [ "$#" -ge 2 ] || { echo "error: incomplete argument: $1" >&2; return 1; }
        case "$1" in
          --version) version=$2 ;;
          --repo) repo=$2 ;;
          --start-ref) start_ref=$2 ;;
          --end-ref) end_ref=$2 ;;
          --base-branch) base_branch=$2 ;;
          --output) output=$2 ;;
        esac
        shift 2
        ;;
      *)
        echo "error: unknown argument: $1" >&2
        return 1
        ;;
    esac
  done

  [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
    echo "error: --version must be an official version in vX.Y.Z form, got: ${version:-<empty>}" >&2
    return 1
  }
  [[ $repo =~ ^[^/[:space:]]+/[^/[:space:]]+$ ]] || {
    echo "error: --repo must be in OWNER/REPO form, got: ${repo:-<empty>}" >&2
    return 1
  }

  local command
  for command in git gh jq awk sort; do
    require_command "$command"
  done
  token=${GITHUB_TOKEN:-${GH_TOKEN:-}}
  [ -n "$token" ] || {
    echo 'error: GITHUB_TOKEN or GH_TOKEN is required to read pull request metadata' >&2
    return 1
  }
  GH_TOKEN=$token
  export GH_TOKEN

  [ -n "$base_branch" ] || base_branch=$(git_cmd branch --show-current)
  git_cmd rev-parse --verify "$end_ref^{commit}" >/dev/null
  if [ -n "$(git_cmd tag --list "$version")" ]; then
    echo "$version already exists; release notes must be committed before the official tag is created" >&2
    return 1
  fi
  [ -n "$start_ref" ] || start_ref=$(find_previous_official_tag "$version" "$end_ref")
  [ -z "$start_ref" ] || git_cmd rev-parse --verify "$start_ref^{commit}" >/dev/null

  if [ -z "$output" ]; then
    local version_number=${version#v}
    local major=${version_number%%.*}
    local remainder=${version_number#*.}
    local minor=${remainder%%.*}
    output="CHANGELOG/CHANGELOG-$major.$minor.md"
  fi

  work_dir=$(mktemp -d "${TMPDIR:-/tmp}/matrixhub-changelog.XXXXXX")
  CHANGELOG_WORK_DIR=$work_dir
  trap 'cleanup_work_dir "${CHANGELOG_WORK_DIR:-}"' EXIT
  trap 'exit 129' HUP
  trap 'exit 130' INT
  trap 'exit 143' TERM
  mkdir -p "$work_dir/entries"
  : > "$work_dir/errors"
  : > "$work_dir/warnings"
  : > "$work_dir/excluded"

  range=$end_ref
  [ -z "$start_ref" ] || range="$start_ref..$end_ref"
  printf 'Collecting %s release notes from %s\n' "$version" "$range"
  collect_pull_numbers "$start_ref" "$end_ref" "$base_branch" "$repo" "$work_dir"
  commit_count=$(line_count "$work_dir/commits")
  printf 'Inspecting %s commits in %s\n' "$commit_count" "$repo"

  while IFS= read -r number; do
    [ -n "$number" ] || continue
    if ! pull_json=$(github_api "repos/$repo/pulls/$number"); then
      echo "error: failed to read pull request #$number" >&2
      return 1
    fi
    process_pull "$pull_json" "$work_dir"
  done < "$work_dir/pull_numbers.sorted"

  while IFS= read -r warning; do
    [ -n "$warning" ] && printf 'warning: %s\n' "$warning" >&2
  done < "$work_dir/warnings"

  if [ -s "$work_dir/errors" ]; then
    echo 'error: Release note metadata validation failed:' >&2
    while IFS= read -r warning; do
      printf -- '- %s\n' "$warning" >&2
    done < "$work_dir/errors"
    return 1
  fi

  render_generated_block "$work_dir" "$repo" "$start_ref" "$version" > "$work_dir/generated.md"
  if [ "$dry_run" = true ]; then
    printf '\n## %s\n\n' "$version"
    cat "$work_dir/generated.md"
    printf '\n'
  else
    upsert_changelog "$output" "$output" "$version" "$work_dir/generated.md" "$work_dir"
    printf 'Updated %s\n' "$output"
  fi

  included_count=$(file_count "$work_dir/entries")
  excluded_count=$(line_count "$work_dir/excluded")
  pull_count=$(line_count "$work_dir/pull_numbers.sorted")
  append_github_output output_path "$output"
  append_github_output start_ref "${start_ref:-repository-root}"
  append_github_output included_count "$included_count"
  append_github_output excluded_count "$excluded_count"
  append_github_output pull_count "$pull_count"

  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      printf '## %s release note collection\n\n' "$version"
      printf -- '- Range: `%s`\n' "${start_ref:-repository-root}..$end_ref"
      printf -- '- Pull requests found: %s\n' "$pull_count"
      printf -- '- Included release notes: %s\n' "$included_count"
      printf -- '- Excluded as NONE/NO: %s\n' "$excluded_count"
      printf -- '- Commits without an associated merged PR: %s\n' \
        "$(grep -c '^no merged PR found' "$work_dir/warnings" || true)"
    } >> "$GITHUB_STEP_SUMMARY"
  fi

  cleanup_work_dir "$work_dir"
  CHANGELOG_WORK_DIR=''
  trap - EXIT HUP INT TERM
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  set -euo pipefail
  main "$@"
fi
