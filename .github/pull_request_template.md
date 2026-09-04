## 📝 概要 / Summary
<!-- このPRの目的や変更内容について簡潔に記述してください。 -->


## 🔗 関連する Issue / Related Issues
- #<!-- Issue番号を記載してください -->

## 📦 変更カテゴリ / Change Category
<!-- 該当するカテゴリにチェックを入れてください。 -->
- [ ] API / HTTP (internal/web)
- [ ] データベース (ent スキーマ / Atlas マイグレーション)
- [ ] CLI (urfave/cli)
- [ ] セキュリティ (認証 / TLS / ACME)
- [ ] オブザーバビリティ (slog / OTel / Prometheus)
- [ ] Terraform / IaC (さくらのクラウド)
- [ ] CI / CD (GitHub Actions / GoReleaser / tagpr)
- [ ] AI スキル (.claude/skills)
- [ ] ドキュメント / テンプレート
- [ ] その他

## 🛠️ 変更内容 / Changes
<!-- どのような変更を加えたかを箇条書きで記載してください。 -->
-

## 🧪 検証チェックリスト / Verification Checklist
<!-- 実施した検証にチェックを入れてください。 -->
- [ ] `make generate` を実行し、`git diff --exit-code` で生成コードに差分がないことを確認した
- [ ] `make lint` が 0 issues で成功することを確認した
- [ ] `make test` が全テスト PASS で成功することを確認した
- [ ] `make build` が正常に完了することを確認した
- [ ] `make license-check` でライセンスヘッダーが適切であることを確認した
- [ ] OpenAPI 変更時: `make openapi-lint` が正常に通過することを確認した
- [ ] Terraform 変更時: `terraform fmt -check`, `validate`, `tflint`, `tfsec` を確認した
- [ ] AI スキル 変更時: `make check` で文法・フロントマターエラーがないことを確認した

## 🚨 注意事項・懸念点 / Notes & Concerns
<!-- 破壊的変更、互換性の懸念、パフォーマンスへの影響などがあれば記述してください。 -->
-
