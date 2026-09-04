---
name: golang-implementation
description: "Go (Golang) の実装プラクティス。エラーハンドリング（%wによるラッピング、errors.Is/Asの利用、Single Handling Rule）、並行処理（goroutineライフサイクル、チャネル所有権、syncプリミティブ、errgroup）、メモリ・ポインタの安全性（nilインタフェイストラップ、スライス/マップのメモリ共有）、Contextの正しい使用方法を適用する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
  openclaw:
    emoji: "⚙"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent
---

> [!IMPORTANT]
> **ガバナンスと変更管理:** 
> 本スキルファイルは組織の標準実装指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびアーキテクトによるレビューを経て行われます。

**Persona:** あなたは Go の堅牢性（Reliability）エンジニアです。すべてのエラー、goroutine、メモリ参照に対して、パニックやバグを事前に防ぐための防衛的（Defensive）なアプローチを取ります。

# Go 実装ベストプラクティス (Go-Specific)

## 1. エラーハンドリング (Error Handling)
- **エラーは必ずチェックする**: `_` でエラーを捨てない。
- **コンテキストの追加 (Wrapping)**: 下層のエラーを上位に返す際は、`fmt.Errorf("context: %w", err)` を用いて元のエラー構造を保持する。
- **エラー値の検査**: エラーの特定には `errors.Is`、特定のカスタムエラー型への変換には `errors.As` を使用する。
- **Single Handling Rule (一回限りのハンドリング)**: エラーは「ログに記録する」か「上位に返す（戻り値として伝搬する）」かのいずれか一方のみを行う。両方行うとログが重複して分析が困難になる。

## 2. 並行処理 (Concurrency)
- **goroutine の明確な終了設計**: goroutine を開始する際は、それがどのように終了するか（Context、キャンセル用チャネル、WaitGroup等）を必ず設計する。終了できない goroutine はメモリリークの原因になる。
- **チャネルの所有権**: チャネルへの書き込みとクローズは、原則としてそのチャネルの「送信側（所有者）」が行う。受信側でのクローズはパニックの原因となる。
- **データ競合の防止**: チャネルで送信するデータはポインタではなく値（コピー）を推奨する。ポインタを送信するとメモリの共有が発生し、データ競合のリスクが高まる。
- **適切な同期プリミティブの選択**:
  - `sync.Mutex` / `sync.RWMutex`: 構造体のフィールドなど、シンプルな排他制御。
  - `x/sync/errgroup`: 複数の並行タスクを実行し、エラーの収集やキャンセルを行いたい場合。

## 3. メモリと安全面 (Safety & Correctness)
- **nil インターフェースの罠**: インターフェース型の変数に「値が nil の具象型ポインタ」を代入すると、インターフェース自体は `nil` と判定されなくなる（`err != nil` が真になる）。
  ```go
  // ❌ 危険な例: インターフェース型を返す関数で、nil ポインタの具象型変数を返す
  func getError() error {
      var err *MyError = nil
      return err // 戻り値は nil ではなくなる
  }
  ```
- **スライスとマップの安全性**: `append` は容量に余裕がある場合、元のバッキング配列を共有する。不要な共有によるデータの書き換えを防ぐため、必要に応じて防衛的コピー（`copy`）を行う。また、nil マップへの書き込みはパニックになるため、必ず `make` で初期化する。

## 4. コンテキスト (Context)
- **シグネチャの第1引数**: I/O や非同期処理を行う関数の第1引数は `ctx context.Context` とする。
- **Context にドメインデータを入れない**: Context にはリクエストIDやトレース情報、タイムアウト情報のみを格納し、アプリケーションのビジネスロジックに影響するデータは通常の引数として渡す。

## 5. ログ出力時の機密情報（Credential/PII）漏洩防止 (SA主導)

ログにパスワード、APIトークン、秘密鍵、クレジットカード番号などの機密情報や個人情報 (PII) が絶対に生のまま記録されないよう、ソフトウェアアーキテクト（SA）とSIRTの連携に基づき、以下の防御策をコード上で徹底します。

### 5.1 静的防御: `slog.LogValuer` インターフェースによる隠蔽
機密情報を保持する専用の型（例：`SecretString`）を定義し、標準ライブラリの `slog.LogValuer` インターフェースを実装します。これにより、オブジェクトがそのままログに渡された場合でも、値が自動的に隠蔽またはハッシュ化されます。

```go
package security

import (
    "crypto/sha256"
    "encoding/hex"
    "log/slog"
)

// SecretString はログ出力時に自動でマスキングされる文字列型です
type SecretString string

// LogValue は slog がログをシリアライズする際に呼び出され、値を隠蔽します
func (s SecretString) LogValue() slog.Value {
    return slog.StringValue("[REDACTED]")
}

// HashableSecret は衝突確認用にSalt付きハッシュ値を出力する型です
type HashableSecret string

func (h HashableSecret) LogValue() slog.Value {
    salt := []byte("system_specific_salt") // 環境変数やVaultから取得
    hasher := sha256.New()
    hasher.Write(salt)
    hasher.Write([]byte(h))
    return slog.StringValue(hex.EncodeToString(hasher.Sum(nil)))
}
```

```go
type Credentials struct {
    Username string
    Password SecretString // ログ出力時に自動で [REDACTED] になる
}

// 誤って構造体を丸ごとログに出力してもパスワードは漏洩しません
slog.Info("login_attempt", "creds", creds)
```

### 5.2 動的防御: `slog.Handler` による一括マスキング
ロガーの初期化時に、特定のキー（`password`、`token`、`secret`、`authorization` など）を持つ属性の値を動的に置換するグローバルフィルタ（`ReplaceAttr`）を設定します。

```go
func NewSecureJSONHandler() slog.Handler {
    return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            keyLower := strings.ToLower(a.Key)
            if keyLower == "password" || keyLower == "token" || keyLower == "secret" || keyLower == "authorization" {
                a.Value = slog.StringValue("[REDACTED]")
            }
            return a
        },
    })
}
```

## 6. ソースファイルのライセンスおよび作成者ヘッダーの付与

新規に Go ソースファイル（`.go`）を作成する、または修正する際は、必ずファイル冒頭に以下の形式で Apache License 2.0 および作成者（`Author`）を示すヘッダーコメントを含めてください。

```go
// Copyright 2026 [Copyright Holder]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: [YOUR_NAME]
```

