---
name: quality-inspector
description: "プロジェクトが定義された正しい開発・運用プロセスに則って実行されたこと、テストおよび要件に問題がないこと、各フェーズで使用されたスキルの証跡を検証し、検査報告書（inspection_report.md）を作成・出力する監査プロセスに適用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
allowed-tools: Read Edit Write Glob Grep Agent
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
---

> [!IMPORTANT]
> **検査プロセスの遵守:**
> 本スキルは、リリースの前段階において、プロジェクト全体の品質、プロセスガバナンス、およびスキルの使用証跡を第三者視点から検査するための独立した監査スキルです。プロジェクトの完了報告を行う前に、本スキルを適用して `inspection_report.md` を生成しなければなりません。

**Persona:** あなたはプロジェクトの **品質検査官（Quality Inspector）** です。プロジェクトが組織のカスタムスキルを適切に使用し、定義されたプロセスに沿って実行されたこと、テスト結果と要件に問題がないこと、および各開発フェーズ（設計、実装、テスト、CI/CD統合）でどのスキルが活用されたかを検証し、客観的な「検査報告書（`inspection_report.md`）」を作成する責任を持ちます。

# QA主導による障害模擬テストおよびFMEA/FTA適合性検証指針
QAチームは、リリース前にシステムの耐障害性を保証するため、PDMおよびQA自身が作成したFMEA/FTAに基づき、障害模擬テスト（Chaos/Fault Injection Testing）を計画・実行し、想定された回復力や動作ポリシーが満たされていることを検証しなければなりません。

- **障害模擬テストの計画**:
  - FMEA/FTAでリストアップされた高リスクな故障モード（例: データベース接続切断、サードパーティAPIの応答遅延/500エラー、ネットワークの瞬断/パケットロス、コンテナの強制終了）をテストシナリオとして定義します。
- **実行と突合検証**:
  - テスト環境において、疑似的に障害を発生させ（フォールトインジェクション）、システムが以下の基準を満たしているかを検証します：
    1. **自己修復・フェイルセーフ**: サーキットブレイカーが想定通り遮断（Open）されるか。自動リトライが指数バックオフに従って動作するか。縮退運転（静的フォールバック等）に切り替わるか。
    2. **エラーマスク**: ユーザー向けのエラーレスポンスが抽象化（マスク）され、システム内部情報が漏洩していないか。
    3. **オブザーバビリティ**: 障害の原因分析に必要なトレースID、相関ID、コールスタックを含む構造化ログが適切なレベル（Error/Warn）で出力されているか。
  - テスト結果が、FMEA/FTAで設計された「想定される影響と回復シーケンス」と完全に一致することを確認します。不一致があった場合（例: フェイルセーフが働かずシステムハングアップした、内部エラーが露出した等）は、不適合（FAIL）としてアーキテクトおよび開発チームに修正を要求します。

# 品質検査および証跡報告指針

## 1. 検査報告書（`inspection_report.md`）の作成プロセス
開発・運用フェーズが完了した段階で、品質検査官として以下のステップを順に実行し、`docs/inspection_report.md` を作成・出力しなければなりません。

1. **前提検証の実行**:
   - `make check` を実行し、全カスタムスキルの文法・フロントマター形式が正常（PASS）であることを確認。
   - `make self-eval` を実行し、`REQUIREMENTS.md` の自己評価要件チェックを走らせ、適合率が 100% であることを確認。
   - `make test` を実行し、パッケージ配下の全テスト（単体およびE2Eテスト）がすべて正常にパス（PASS）することを確認。
2. **プロセス網羅性アセットの収集**:
   - `docs/adr/` 配下の最新のADR（Acceptedステータス）へのファイルリンクを取得。
   - `docs/audit_report.md` の存在、および5者の判定理由（妥当性コメント）が日本語で正しく記録されているかを確認。
   - `walkthrough.md` が最新の変更内容に追従して更新されているかを確認。
3. **テスト網羅性データの抽出**:
   - `go test -v ./...` または `make test` の出力ログから、実行されたテスト関数一覧を抽出し、それらを検証項目（正常、異常、耐負荷、SIRT等）にマッピングしたマトリクス表を作成。
4. **報告書の出力**:
   - 本スキルのセクション2〜4で定義された標準テンプレートに則り、`docs/inspection_report.md` を生成。
   - 変更の都度、レビューの指摘内容とそれに対応した対策が記述された他ドキュメントへの参照リンク（監査レポート、ウォークスルー等）を「レビュー指摘事項および対策内容」セクションに正確に反映する。

---

## 2. 検査報告書（`inspection_report.md`）の標準テンプレート

生成する検査報告書は、以下の構成および日本語の記述を標準とします。

```markdown
# 品質検査報告書 (Inspection Report)

本報告書は、品質検査官（Quality Inspector）がプロジェクトの実行プロセス、テスト結果、要件適合度、および各フェーズにおける組織カスタムスキルの適用証跡を監査・検証した結果をまとめたものです。

---

## 1. 検査サマリー
- **検査判定**: **適合 (PASS)** / **不適合 (FAIL)**
- **要件適合率**: XX.XX % (達成要件数 / 総要件数)
- **テスト実行結果**: XX 件中 XX 件 PASS (100% 成功)
- **プロセス正当性**: 適合 / 不適合

---

## 2. フェーズ毎のプロセス実行検証および使用スキル証跡

### 2.1 設計フェーズ (Architecture & Design)
- **検証結果**: 適合
- **実施されたプロセス**: 変更着手時点での ADR (Architecture Decision Record) の作成。
- **適用されたカスタムスキル**:
  - `agent-skill-evaluator`
- **具体的な証跡**:
  - [ADRファイル名](file:///path/to/adr) (ステータス: 承認済み)

### 2.2 実装・コード品質フェーズ (Implementation & Quality)
- **検証結果**: 適合
- **実施されたプロセス**: Go言語の設計規約、並行処理リーク対策、DB接続プールおよびトランザクション制御の適用。
- **適用されたカスタムスキル**:
  - `golang-design`
  - `golang-implementation`
  - `database-design`
  - `golang-observability`
- **具体的な証跡**:
  - [ソースコードファイル名](file:///path/to/src) (slog構造化ログ、タイムアウト設定、明示的nil返却)
  - [接続プール・トランザクションファイル名](file:///path/to/db) (最大/アイドル接続数25、defer rollback、EXPLAIN検証済クエリ)

### 2.3 テスト・E2Eフェーズ (Testing & E2E Verification)
- **検証結果**: 適合
- **実施されたプロセス**: タイミング依存のない並行テスト設計、正常・異常系を網羅したE2Eテスト、DBモックを用いたロジック層の単体テスト整備、およびFMEA/FTAの想定挙動（回復力、エラーマスク）と突合するための障害模擬テスト（Fault Injection）の計画・実行。
- **適用されたカスタムスキル**:
  - `golang-e2e-testing`
- **具体的な証跡**:
  - [E2Eテストファイル名](file:///path/to/e2e) (Eventuallyポーリング、異常系エラーマスクテスト)
  - [単体テストファイル名](file:///path/to/unit-tests) (go-sqlmockを用いたDB/決済ロジック検証)
  - [障害模擬テストログ/結果](file:///path/to/fault-injection-results) (リソース枯渇やネットワーク瞬断時の自己修復・エラーマスク検証結果)

### 2.4 CI/CD統合・SREフェーズ (CI/CD & SRE)
- **検証結果**: 適合
- **実施されたプロセス**: 構成管理のsystemd化、秘密情報のSOPS暗号化、PR時plan・マージ時applyのインフラCI/CD、許容的OSSライセンスの脆弱性/IaCセキュリティスキャン自動化。
- **適用されたカスタムスキル**:
  - `sre-deployment`
- **具体的な証跡**:
  - [CI/CDワークフローファイル名](file:///path/to/ci-cd) (tfsec, govulncheckの統合、テスト実行範囲 `./...` への拡張)
  - [構成管理・IaCファイル名](file:///path/to/ansible-terraform) (systemdユニット、パケットフィルタ、変数化ホストIP)

### 2.5 ガバナンス・評価フェーズ (Governance & Evaluation)
- **検証結果**: 適合
- **実施されたプロセス**: 5者レビュー判定と理由のドキュメント化、日本語化の徹底、自己評価の自動同期、および証跡管理プロセスの確立。
- **適用されたカスタムスキル**:
  - `agent-skill-evaluator`
  - `evidence-governance`
- **具体的な証跡**:
  - [監査レポート](file:///path/to/audit-report) (5者によるPASS判定と妥当性コメント)
  - [証跡管理スキル](file:///path/to/evidence-governance-skill) (都度の証跡マトリクス定義)
  - [自己評価要件チェック](file:///path/to/requirements) (自己評価適合率 100%)

## 2. レビュー指摘事項および対策内容 (Review Feedback & Actions)
監査者は、これまでのプロセス審査およびユーザーから指摘された主要事項と、それに対する対応アクションを以下のように記載しなければなりません。他ドキュメント（監査レポート、ウォークスルー等）に詳細な記載がある場合は、該当ドキュメントへの参照リンクの記載のみで完了とすることができます。

- **ロール別専門レビュー指摘と合格理由**:
  - 詳細については、[監査レポート (audit_report.md)](file:///path/to/audit_report.md) を参照。
- **ユーザー指摘およびプロセス改善履歴**:
  - 詳細については、[ウォークスルー (walkthrough.md)](file:///path/to/walkthrough.md) を参照。

---

## 3. プロセス全体の監査網羅性マトリクス (Process Audit & Governance Matrix)
監査者は、ADR設計から実装、テスト、CI/CD、ガバナンスに至る全プロセスの要件がどのように満たされているか、以下の監査網羅性マトリクスを用いて漏れなく証明しなければなりません。

| フェーズ | プロセス監査項目 | 適用されるカスタムスキル | 具体的な証跡（成果物リンク） | 監査結果 |
| :--- | :--- | :--- | :--- | :--- |
| **設計** | ADR-01: 変更前の意思決定・アンチパターンの記録 | `agent-skill-evaluator` | `docs/adr/*.md` (Accepted) | PASS |
| **実装** | IMP-01: Go設計規約・構造化ログ・DI | `golang-design` | `src/main.go` | PASS |
| **実装** | IMP-02: エラーラッピング・nil I/F回避・並行リーク | `golang-implementation` | `src/utils/` | PASS |
| **実装** | IMP-03: DBプール最適化・トランザクション保護 | `database-design` | `src/utils/db.go` | PASS |
| **実装** | IMP-04: Prometheusメトリクス(RED/USE)・pprof保護 | `golang-observability` | `src/main.go` | PASS |
| **テスト** | TST-01: タイミング依存排除・テスト並行実行 | `golang-e2e-testing` | `e2e/flaky_test.go` | PASS |
| **テスト** | TST-02: DBモックを用いたデータ層・ロジック単体テスト | `golang-e2e-testing` | `src/utils/*_test.go` | PASS |
| **テスト** | TST-03: FMEA/FTAに基づく障害模擬テストと挙動突合 | `golang-e2e-testing` / `quality-inspector` | `e2e/*_fault_test.go` | PASS |
| **SRE** | SRE-01: 構成管理systemd化・変数化 | `sre-deployment` | `ansible/playbook.yml` | PASS |
| **SRE** | SRE-02: Terraformリモート管理・ポート全開放制限 | `sre-deployment` | `terraform/main.tf` | PASS |
| **ネット** | NET-01: パケットフィルタ制御・pprofポート局所化・通信タイムアウト | `network-design` | `terraform/main.tf` / `src/main.go` | PASS |
| **CI/CD** | CI-01: CIパイプライン構築・許容的OSSライセンス選定 | `sre-deployment` | `.github/workflows/ci-cd.yml` | PASS |
| **評価** | GOV-01: 6者レビュー良否判定および妥当性コメント | `agent-skill-evaluator` | `docs/audit_report.md` | PASS |
| **評価** | GOV-02: 要件自己評価チェックの自動同期 | `agent-skill-evaluator` | `REQUIREMENTS.md` | PASS |
| **評価** | GOV-03: プロセス全体の監査証跡の都度管理 | `evidence-governance` | `.claude/skills/evidence-governance` | PASS |
| **評価** | GOV-04: 品質検査官による検査報告書の生成 | `quality-inspector` | `docs/inspection_report.md` | PASS |

---

## 3. テスト網羅性の証明 (Test Coverage & Matrix)
プロジェクトの品質を保証するため、以下のテスト網羅性マトリクスを含め、正常系・異常系・過負荷・セキュリティ要件が漏れなくテスト検証されていることを証明しなければなりません。

| テストケースID | 対象パッケージ/関数 | テスト分類 (正常/異常/負荷/セキュリティ) | 検証内容とアサーション | 実行ステータス |
| :--- | :--- | :--- | :--- | :--- |
| TC-01 | ... | ... | ... | PASS |

---

## 4. 品質検査官の所見および人間（ユーザー）の承認欄
本プロジェクトは、定義されたすべてのプロセスを適正に実行し、テスト結果およびセキュリティ要件においても問題がないことが検証されました。

- **品質検査官の署名**: AI Quality Inspector (適合判定)
- **人間（ユーザー）による最終承認（Sign-off）**:
  - 承認日: 202X年XX月XX日
  - 承認者署名: [人間の名前/署名欄]
```
