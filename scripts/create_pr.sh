#!/bin/bash
# Copyright 2026 [Copyright Holder]
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Author: [YOUR_NAME]

# scripts/create_pr.sh
# GitHub CLI を使用して、指定されたタイトルと内容でPRを作成するスクリプト。
# プッシュ自体は行わず、リモートへのブランチ存在確認を行った上で実行します。

set -euo pipefail

# PATHの解決 (macOSのHomebrew対策)
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

# 1. 依存関係 (gh) の確認
if ! command -v gh &> /dev/null; then
	echo "Error: github-cli (gh) is not installed. Please install it first." >&2
	exit 1
fi

# 2. 認証状態の確認
if ! gh auth status &> /dev/null; then
	echo "Error: gh is not authenticated. Please run 'gh auth login' first." >&2
	exit 1
fi

# 3. 現在のブランチ名の取得
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
	echo "Error: Cannot create a PR from '$CURRENT_BRANCH' branch." >&2
	exit 1
fi

# 4. リモートブランチの存在確認
if ! git ls-remote --exit-code --heads origin "$CURRENT_BRANCH" &> /dev/null; then
	echo "Error: Branch '$CURRENT_BRANCH' does not exist on remote 'origin'." >&2
	echo "Please push your branch manually first: git push -u origin $CURRENT_BRANCH" >&2
	exit 1
fi

# 引数の処理
TITLE=""
BODY_FILE=""
DRAFT_FLAG=""

while [[ $# -gt 0 ]]; do
	case $1 in
		--title)
			TITLE="$2"
			shift 2
			;;
		--body-file)
			BODY_FILE="$2"
			shift 2
			;;
		--draft)
			DRAFT_FLAG="--draft"
			shift
			;;
		*)
			echo "Unknown option: $1" >&2
			exit 1
			;;
	esac
done

if [ -z "$TITLE" ]; then
	echo "Error: --title is required." >&2
	exit 1
fi

if [ -z "$BODY_FILE" ] || [ ! -f "$BODY_FILE" ]; then
	echo "Error: --body-file is required and must be a valid file." >&2
	exit 1
fi

# PRの作成
echo "Creating Pull Request on GitHub..."
gh pr create --title "$TITLE" --body-file "$BODY_FILE" $DRAFT_FLAG
