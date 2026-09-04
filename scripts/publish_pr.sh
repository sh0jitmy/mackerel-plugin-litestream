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

# scripts/publish_pr.sh
# 開発者がローカルで実行し、検証 (lint, test) -> git push -> gh pr create を一貫して行うスクリプト。

set -euo pipefail

# PATHの解決 (macOSのHomebrew対策)
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

# 1. 依存関係の確認
if ! command -v gh &> /dev/null; then
	echo "Error: github-cli (gh) is not installed. Please install it first." >&2
	exit 1
fi

if ! gh auth status &> /dev/null; then
	echo "Error: gh is not authenticated. Please run 'gh auth login' first." >&2
	exit 1
fi

CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
	echo "Error: Cannot push and create PR from '$CURRENT_BRANCH' branch." >&2
	exit 1
fi

# 2. ローカル検証の実行
echo "=========================================================="
echo "==> 1. Running local validations (fmt, lint, test, license)..."
echo "=========================================================="

echo "--> Running: make fmt"
make fmt

echo "--> Running: make lint"
make lint

echo "--> Running: make test"
make test

echo "--> Running: make license-check"
make license-check

echo "=========================================================="
echo "SUCCESS: All validations passed!"
echo "=========================================================="

# 3. リモートプッシュ
echo "==> 2. Pushing local branch '$CURRENT_BRANCH' to remote origin..."
git push -u origin "$CURRENT_BRANCH"

# 4. PR作成
echo "=========================================================="
echo "==> 3. Creating Pull Request on GitHub..."
echo "=========================================================="

# ブラウザを使用してPR作成画面を立ち上げることを優先します (テンプレートが自動適用されます)
if gh pr create --web; then
	echo "PR creation page opened in your browser."
else
	echo "Falling back to interactive CLI PR creation..."
	gh pr create
fi
