#!/bin/bash

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

# usage:
#  GH_TOKEN=${{ github.token }} LABEL_FEATURE="release/feature-new" LABEL_CHANGED="release/feature-changed" LABEL_BUG="release/bug" PROJECT_REPO="xxx/xxx" changelog.sh  ./ v0.3.6
#  GH_TOKEN=${{ github.token }} LABEL_FEATURE="release/feature-new" LABEL_CHANGED="release/feature-changed" LABEL_BUG="release/bug" PROJECT_REPO="xxx/xxx" changelog.sh  ./ v0.3.6 v0.3.5


CURRENT_DIR_PATH=$(cd `dirname $0`; pwd)
PROJECT_ROOT_PATH="${CURRENT_DIR_PATH}/.."

set -x
set -o errexit
set -o nounset
set -o pipefail

# optional
OUTPUT_DIR=${1}
# required
DEST_TAG=${2}
# optional
START_TAG=${3:-""}


LABEL_FEATURE=${LABEL_FEATURE:-"release/feature-new"}
LABEL_CHANGED=${LABEL_CHANGED:-"release/feature-changed"}
LABEL_BUG=${LABEL_BUG:-"release/bug"}
PROJECT_REPO=${PROJECT_REPO:-""}
[ -n "$PROJECT_REPO" ] || { echo "miss PROJECT_REPO"; exit 1 ; }
[ -n "${GH_TOKEN}" ] || { echo "error, miss GH_TOKEN"; exit 1 ; }

( cd ${OUTPUT_DIR} ) || { echo "error, OUTPUT_DIR '${OUTPUT_DIR}' is not a valid directory" ; exit 1 ; }
OUTPUT_DIR=$(cd ${OUTPUT_DIR}; pwd)
echo "generate changelog to directory ${OUTPUT_DIR}"

# change to root for git cli
cd ${PROJECT_ROOT_PATH}

#============================
# Find the previous tag for changelog range
# If START_TAG is not provided, find the previous official version tag from git
if [ -z "${START_TAG}" ] ; then
    echo "-------------- find previous tag"
    # List all official version tags sorted by version, find the one just before DEST_TAG
    PREV_TAG=$(git tag --sort=version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | grep -B1 "^${DEST_TAG}$" | head -1)
    if [ "$PREV_TAG" = "$DEST_TAG" ] || [ -z "$PREV_TAG" ]; then
        # No previous tag, this is the first release
        START_TAG=""
        echo "No previous tag found, this is the first release"
    else
        START_TAG="$PREV_TAG"
        echo "Previous tag: ${START_TAG}"
    fi
fi

# Use v0.0.0 as display name for first release (for filename and content)
DISPLAY_START_TAG="${START_TAG:-v0.0.0}"

echo "-------------- check tags "
echo "DEST_TAG=${DEST_TAG}"
echo "START_TAG=${DISPLAY_START_TAG}"

# Get commits in range
ALL_COMMIT=""
if [ -z "${START_TAG}" ]; then
    # First release: all commits up to DEST_TAG
    ALL_COMMIT=$(git log ${DEST_TAG} --reverse --oneline | awk '{print $1}' | tr '\n' ' ') \
        || { echo "error, failed to get commits for tag ${DEST_TAG}" ; exit 1 ; }
else
    ALL_COMMIT=$(git log ${START_TAG}..${DEST_TAG} --reverse --oneline | awk '{print $1}' | tr '\n' ' ') \
        || { echo "error, failed to get commits for range ${START_TAG}..${DEST_TAG}" ; exit 1 ; }
fi
echo "ALL_COMMIT: ${ALL_COMMIT}"

TOTAL_COUNT="0"
PR_LIST=""
#
FEATURE_PR=""
CHANGED_PR=""
FIX_PR=""
for COMMIT in ${ALL_COMMIT} ; do
  COMMIT_INFO=` curl --retry 10 -s -H "Accept: application/vnd.github.groot-preview+json" -H "Authorization: Bearer ${GH_TOKEN}" "https://api.github.com/repos/${PROJECT_REPO}/commits/${COMMIT}/pulls" `
  PR=` jq -r '.[].number' <<< "${COMMIT_INFO}" `
  [ -n "${PR}" ] || { echo "warning, failed to find PR number for commit ${COMMIT} " ; echo "${COMMIT_INFO}" ; echo "" ; continue ; }
  if grep " ${PR} " <<< " ${PR_LIST} " &>/dev/null ; then
    continue
  else
    PR_LIST+=" ${PR} "
  fi
  (( TOTAL_COUNT+=1 ))
	INFO=` gh pr view ${PR}  `
	TITLE=` grep -E "^title:[[:space:]]+" <<< "$INFO" | sed -E 's/title:[[:space:]]+//' `
	LABELS=` grep -E "^labels:[[:space:]][^\[]" <<< "$INFO" | sed -E 's/labels://' | tr ',' ' ' ` || true
	#
	PR_URL="https://github.com/${PROJECT_REPO}/pull/${PR}"
	#
	if grep -E "[[:space:]]${LABEL_FEATURE}[[:space:]]" <<< " ${LABELS} " &>/dev/null ; then
	  echo "get new feature PR ${PR}"
		FEATURE_PR+="* ${TITLE} : [PR ${PR}](${PR_URL})
"
	elif grep -E "[[:space:]]${LABEL_CHANGED}[[:space:]]" <<< " ${LABELS} " &>/dev/null ; then
	  echo "get changed feature PR ${PR}"
		CHANGED_PR+="* ${TITLE} : [PR ${PR}](${PR_URL})
"
	elif grep -E "[[:space:]]${LABEL_BUG}[[:space:]]" <<< " ${LABELS} " &>/dev/null ; then
	  echo "get bug fix PR ${PR}"
		FIX_PR+="* ${TITLE} : [PR ${PR}](${PR_URL})
"
	fi
done
#---------------------
echo "generate changelog md"
FILE_CHANGELOG="${OUTPUT_DIR}/changelog_from_${DISPLAY_START_TAG}_to_${DEST_TAG}.md"
echo > ${FILE_CHANGELOG}
echo "# ${DEST_TAG}" >> ${FILE_CHANGELOG}
echo "Welcome to the ${DEST_TAG} release of MatrixHub!" >> ${FILE_CHANGELOG}
echo "Compared with version:${DISPLAY_START_TAG}, version:${DEST_TAG} has the following updates." >> ${FILE_CHANGELOG}
echo "" >> ${FILE_CHANGELOG}
echo "***" >> ${FILE_CHANGELOG}
echo "" >> ${FILE_CHANGELOG}
#
if [ -n "${FEATURE_PR}" ]; then
    echo "## New Feature" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
    while read LINE ; do
      echo "${LINE}" >> ${FILE_CHANGELOG}
      echo "" >> ${FILE_CHANGELOG}
    done <<< "${FEATURE_PR}"
    echo "***" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
fi
#
if [ -n "${CHANGED_PR}" ]; then
    echo "## Changed Feature" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
    while read LINE ; do
      echo "${LINE}" >> ${FILE_CHANGELOG}
      echo "" >> ${FILE_CHANGELOG}
    done <<< "${CHANGED_PR}"
    echo "***" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
fi
#
if [ -n "${FIX_PR}" ]; then
    echo "## Fix" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
    while read LINE ; do
      echo "${LINE}" >> ${FILE_CHANGELOG}
      echo "" >> ${FILE_CHANGELOG}
    done <<< "${FIX_PR}"
    echo "***" >> ${FILE_CHANGELOG}
    echo "" >> ${FILE_CHANGELOG}
fi
#
echo "## Total " >> ${FILE_CHANGELOG}
echo "" >> ${FILE_CHANGELOG}
echo "Pull request number: ${TOTAL_COUNT}" >> ${FILE_CHANGELOG}
echo "" >> ${FILE_CHANGELOG}
if [ -n "${START_TAG}" ]; then
    echo "[ Commits ](https://github.com/${PROJECT_REPO}/compare/${START_TAG}...${DEST_TAG})" >> ${FILE_CHANGELOG}
else
    echo "[ Commits ](https://github.com/${PROJECT_REPO}/commits/${DEST_TAG})" >> ${FILE_CHANGELOG}
fi
echo "--------------------"
echo "generate changelog to ${FILE_CHANGELOG}"
