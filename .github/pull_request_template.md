## 📝 概要 / Summary
<!-- このPRの目的や変更内容について簡潔に記述してください。 -->


## 🔗 関連する Issue / Related Issues
- #<!-- Issue番号を記載してください -->

## 📦 変更カテゴリ / Change Category
<!-- 該当するカテゴリにチェックを入れてください。 -->
- [ ] メトリクス / プラグイン機能 (`internal/litestream`)
  - [ ] Prometheus メトリクス収集 (レプリケーション / WAL)
  - [ ] スナップショット / ストレージ監視
  - [ ] リストア整合性検証 (Restore Check)
  - [ ] systemd サービス状態監視
- [ ] CLI / 設定フラグ (`main.go`)
- [ ] CI / CD / リリース (`GitHub Actions`, `GoReleaser`, `tagpr`)
- [ ] テスト / 検証環境 (`docker-compose`, ユニットテスト, モック)
- [ ] ドキュメント / 仕様書 (`README.md`, `docs/`)
- [ ] 依存関係 / ツール設定 (`go.mod`, `Makefile` 等)
- [ ] その他

## 🛠️ 変更内容 / Changes
<!-- どのような変更を加えたかを箇条書きで記載してください。 -->
- 

## 🧪 検証チェックリスト / Verification Checklist
<!-- 実施した検証にチェックを入れてください。 -->
- [ ] `make lint` が 0 issues で成功することを確認した
- [ ] `make test` で全テストが PASS し、カバレッジ基準（>= 95%）を満たすことを確認した
- [ ] `make build` でバイナリ（`bin/mackerel-plugin-litestream`）が正常に生成されることを確認した
- [ ] `make license-check` でソースコードのライセンスヘッダーが適切であることを確認した
- [ ] GoReleaser 変更時: `goreleaser check` (または `go run github.com/goreleaser/goreleaser/v2@latest check`) が成功することを確認した
- [ ] 動作確認: ローカルまたは Docker 検証環境でメトリクス出力 / プラグイン実行を確認した（該当する場合）

## 🚨 注意事項・懸念点 / Notes & Concerns
<!-- メトリクスキーの変更、破壊的変更、Litestream CLI/S3 API 互換性、Mackerel メトリクス課金への影響などがあれば記述してください。 -->
- 
