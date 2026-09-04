---
name: sakuracloud-terraform
description: "さくらのクラウド (sacloud/sakura v3) を用いた Terraform / IaC 設計、環境構築、モジュール設計、CI/CD 設定の適用およびコードレビューを行う際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.0.0"
allowed-tools: Read Edit Write Glob Grep Agent AskUserQuestion
---

# Terraform Skill for Sakura Cloud (sacloud/sakura v3)

さくらのクラウドを対象とした Terraform（IaC）コードの生成、修正、およびレビューを行う際は、以下の設計・運用原則を**必ず**守ってください。

---

## 1. 基本原則

* **インフラ定義のみに特化する**: Terraform は純粋なリソースの作成・構成管理のみを担い、アプリケーションレイヤーのロジックを Terraform コード内に書かないでください。
* **OS設定の埋め込み禁止**: サーバー内の詳細なOS設定やソフトウェア構成などは、Ansible などの構成管理ツールやコンテナのレイヤーで行うべきであり、Terraform コード内に直接組み込まないでください。
* **以下のリソースおよびプロビショナーの使用禁止**:
  * ❌ `provisioner` (local-exec / remote-exec) の使用禁止
  * ❌ `null_resource` の使用禁止
  * ❌ 大量のシェルスクリプトを埋め込む行為の禁止
* **不明点がある場合は質問する**: 要件に曖昧な点がある場合は、勝手にコードを書かずに、以下の点についてユーザーに質問・確認してください：
  * 対象となるさくらのクラウドのサービス
  * 対象の環境数（例: dev, stg, prod）
  * ネットワーク構成（閉域接続、またはインターネット接続の有無）
  * 使用する Zone 構成
  * State の分割単位
  * 適用する CI/CD 方式
  * 機密情報（Secret）の管理方法

---

## 2. Terraform / Provider バージョン固定

* **Terraform バージョン指定**:
  必ず `~> 1.13` に固定し、最小限のバージョン制約を付与します。
  ```hcl
  terraform {
    required_version = "~> 1.13"
  }
  ```
* **Provider 指定（sacloud/sakura v3必須）**:
  必ず `sacloud/sakura` (v3) を使用します。新規構築において古い `sacloud/sakuracloud` (v2) は利用しないでください。また、プロバイダーバージョンは `~> 3.12` に固定します。
  ```hcl
  terraform {
    required_providers {
      sakura = {
        source  = "sacloud/sakura"
        version = "~> 3.12"
      }
    }
  }
  ```
  * ❌ バージョン未指定の禁止
  * ❌ `version = "latest"` や `version = ">= 3"` のような、将来のマイナー・メジャー更新による破壊的変更を許容する記述 of 禁止

---

## 3. Provider設定とCredential管理

* **Provider 定義の配置先**:
  Provider 定義（`provider "sakura" {}`）は、環境配下のディレクトリにのみ配置します。
  * 🟢 良い例: `environments/prod/provider.tf`
  * ❌ モジュール（`modules/`）の内部での provider 定義の記述禁止
* **認証情報（Credential）管理**:
  トークンやシークレットなどの認証情報を Terraform コードに記述、またはハードコードしてはなりません。
  * ❌ 禁止例:
    ```hcl
    provider "sakura" {
      token  = "xxx"
      secret = "xxx"
    }
    ```
    または `variable "token"`, `variable "secret"` を tfvars などで直接定義して管理する行為の禁止。
  * 🟢 推奨:
    環境変数 `SAKURA_ACCESS_TOKEN`, `SAKURA_ACCESS_TOKEN_SECRET`, `SAKURA_ZONE` を使用。
    あるいは、さくらのクラウドの `profile` を利用する。
    CI/CD 実行時には、GitHub Secrets 等を環境変数経由で注入する。

---

## 4. ディレクトリ構成と State 設計

### 4.1 ディレクトリ構成
Terraform プロジェクトは原則として以下の構成をとります。

```
terraform/
├── modules/          # カテゴリー別モジュール
│   ├── Application Integration/
│   │   └── (addon などのモジュール)
│   ├── Computing/
│   │   └── (server, private_host などのモジュール)
│   ├── Container and Image/
│   │   └── (container_registry などのモジュール)
│   ├── Database/
│   │   └── (database, nosql などのモジュール)
│   ├── Misc/
│   │   └── (script, ssh_key などのモジュール)
│   ├── Monitoring/
│   │   └── (simple_monitor などのモジュール)
│   ├── Networking/
│   │   └── (switch, internet, dns などのモジュール)
│   ├── Platform/
│   │   └── (apprun などのモジュール)
│   ├── Security/
│   │   └── (packet_filter, secret_manager などのモジュール)
│   └── Storage and Data/
│       └── (disk, archive などのモジュール)
└── environments/     # 各環境（環境ごとにStateを分離）
    ├── dev/
    ├── stg/
    └── prod/
```

### 4.2 State設計
* **責務ごとの State 分離**:
  更新頻度、ライフサイクル、担当チーム、障害時の影響範囲に応じて、State（バックエンド）を細かく分割します。
  * 例: `network/`, `compute/`, `storage/`, `monitoring/` などのディレクトリごとに State を分離する。
* **Local State の禁止**:
  実稼働環境やチーム開発においては Local State を禁止し、リモートバックエンド（S3互換オブジェクトストレージや Terraform Cloud など）を必ず構成します。

---

## 5. Module設計 と リソース配置ルール

* **リソースの配置ルール**:
  各インフラリソース（`sakura_*`）は、プロバイダドキュメントに記載された `subcategory`（サブカテゴリー）に基づいて、以下の対応表の通り `modules/` 配下のサブディレクトリに分類して配置してください。

| カテゴリー (Subcategory) | 対象となるさくらクラウド用リソース (`sakura_*`) |
| :--- | :--- |
| **Computing** | `sakura_server`, `sakura_private_host`, `sakura_auto_scale` |
| **Storage and Data** | `sakura_disk`, `sakura_archive`, `sakura_cdrom`, `sakura_dedicated_storage`, `sakura_nfs`, `sakura_object_storage_bucket`, `sakura_object_storage_bucket_cors`, `sakura_object_storage_bucket_encryption_config`, `sakura_object_storage_bucket_replication_config`, `sakura_object_storage_bucket_versioning`, `sakura_object_storage_object`, `sakura_object_storage_permission` |
| **Networking** | `sakura_switch`, `sakura_vswitch`, `sakura_internet`, `sakura_bridge`, `sakura_local_router`, `sakura_vpn_router`, `sakura_dns`, `sakura_dns_record`, `sakura_gslb`, `sakura_subnet`, `sakura_ipv4_ptr`, `sakura_webaccel`, `sakura_webaccel_acl`, `sakura_webaccel_activation`, `sakura_webaccel_certificate`, `sakura_seg` |
| **Security** | `sakura_packet_filter`, `sakura_packet_filter_rules`, `sakura_kms`, `sakura_cloudhsm`, `sakura_cloudhsm_client`, `sakura_cloudhsm_license`, `sakura_cloudhsm_peer`, `sakura_secret_manager`, `sakura_secret_manager_secret`, `sakura_security_control_activation`, `sakura_security_control_automated_action`, `sakura_security_control_evaluation_rule` |
| **Database** | `sakura_database`, `sakura_database_read_replica`, `sakura_enhanced_db`, `sakura_nosql`, `sakura_nosql_additional_nodes`, `sakura_ondemand_db` |
| **Container and Image**| `sakura_container_registry` |
| **Monitoring** | `sakura_simple_monitor`, `sakura_monitoring_suite_alert_log_measure_rule`, `sakura_monitoring_suite_alert_notification_routing`, `sakura_monitoring_suite_alert_notification_target`, `sakura_monitoring_suite_alert_project`, `sakura_monitoring_suite_alert_rule`, `sakura_monitoring_suite_dashboard`, `sakura_monitoring_suite_log_routing`, `sakura_monitoring_suite_log_storage`, `sakura_monitoring_suite_log_storage_access_key`, `sakura_monitoring_suite_metric_routing`, `sakura_monitoring_suite_metric_storage`, `sakura_monitoring_suite_metric_storage_access_key`, `sakura_monitoring_suite_trace_storage`, `sakura_monitoring_suite_trace_storage_access_key`, `sakura_simple_notification_destination`, `sakura_simple_notification_group`, `sakura_simple_notification_routing` |
| **Application Integration** | `sakura_addon_ai`, `sakura_addon_cdn`, `sakura_addon_datalake`, `sakura_addon_ddos`, `sakura_addon_dwh`, `sakura_addon_etl`, `sakura_addon_query`, `sakura_addon_search`, `sakura_addon_streaming`, `sakura_addon_waf`, `sakura_apigw_cert`, `sakura_apigw_domain`, `sakura_apigw_group`, `sakura_apigw_route`, `sakura_apigw_service`, `sakura_apigw_subscription`, `sakura_apigw_user`, `sakura_simple_mq`, `sakura_eventbus_process_configuration`, `sakura_eventbus_schedule`, `sakura_eventbus_trigger` |
| **Platform** | `sakura_apprun_dedicated_application`, `sakura_apprun_dedicated_auto_scaling_group`, `sakura_apprun_dedicated_certificate`, `sakura_apprun_dedicated_cluster`, `sakura_apprun_dedicated_lb`, `sakura_apprun_dedicated_version`, `sakura_apprun_shared`, `sakura_iam_auth`, `sakura_iam_folder`, `sakura_iam_group`, `sakura_iam_organization_id_policy`, `sakura_iam_policy`, `sakura_iam_project`, `sakura_iam_project_apikey`, `sakura_iam_service_principal`, `sakura_iam_sso`, `sakura_iam_user`, `sakura_iam_user_provisioning` |
| **Misc** | `sakura_ssh_key`, `sakura_script`, `sakura_icon`, `sakura_auto_backup`, `sakura_workflows`, `sakura_workflows_revision_alias`, `sakura_workflows_subscription` |

* **モジュールは単一責任（責務単位）**:
  モジュールは機能・責務ごとに作成し、必ず上記のカテゴリー別ディレクトリの下に配置します。
  * 🟢 良い例: `modules/Computing/server/` にサーバー構成モジュールを配置し、`modules/Storage and Data/disk/` にディスク構成モジュールを配置。
  * ❌ 禁止: `modules/infrastructure/` のような「全部入り巨大モジュール」
  * ❌ 禁止: `server` + `disk` + `switch` + `packet_filter` をすべて1つのモジュールにまとめてカテゴリ配下に置かない設計
  * ❌ 禁止: モジュールのネストを3段以上にすること（最大2階層まで）

---

## 6. さくらクラウド固有設計

* **Server と Disk のライフサイクル分離**:
  さくらのクラウドにおいて、サーバーとディスクは物理的に別のオブジェクトです。これらを別々のリソースとして定義し、アタッチメントを介して接続することで、サーバー破棄時にもデータディスクが消失しないよう安全に設計してください。
  * 🟢 推奨リソース: `sakura_server`, `sakura_disk`, `sakura_disk_attachment`
* **Packet Filter は独立したモジュールにする**:
  ファイアウォール（Packet Filter）の設定はセキュリティの監査対象となるため、個別のモジュールとして独立して管理します。
* **Switch/Router の AWS VPC 扱い禁止**:
  さくらのクラウドにおけるスイッチやルーターの挙動は、AWS の VPC とは異なります。固有のルーティング仕様に合わせ、IPアドレスプールやゲートウェイの割り振りを正確に実装してください。
* **Zone のハードコード禁止**:
  Zone（`tk1a`, `tk1b`, `is1a`, `is1b` など）はハードコードせず、変数として受け取るようにします。また、バリデーションを設定して不正な Zone が指定されないようにしてください。
  ```hcl
  variable "zone" {
    type = string
    validation {
      condition     = contains(["tk1a", "tk1b", "is1a", "is1b"], var.zone)
      error_message = "Unsupported zone. Choose from tk1a, tk1b, is1a, or is1b."
    }
  }
  ```

---

## 7. Resource・Naming・Security・Lifecycle

* **リソース生成**:
  * ❌ `count` の使用禁止（配列インデックスによるリソース管理は、要素の順序変更時に予期せぬ destroy を引き起こすため）。
  * 🟢 `for_each` の使用を優先。一意のキー（ホスト名など）を持つマップ型変数で定義します。
* **ネーミングルール**:
  原則として `{project}-{env}-{resource}-{sequence}` の形式に従います。
  * 例: `monitor-prod-server-001`
* **共通タグ（Local変数の定義）**:
  すべてのインフラリソースに適用できるよう、`common_tags` を定義してください。
  ```hcl
  locals {
    common_tags = {
      Environment = var.env
      Project     = var.project
      ManagedBy   = "terraform"
    }
  }
  ```
* **セキュリティ要件**:
  * 機密情報（Secret）を `tfvars` に直接記述しない。
  * パケットフィルターは最小限のアクセス許可（最小権限の原則）にする。
  * パブリックインターネットからの接続を許容する場合は、必ずその必要性をコード内のコメントに明記する。
  * 初期パスワードなどをコード内にハードコードしない。
* **ライフサイクルルール**:
  * `lifecycle.ignore_changes` を使用する場合は、なぜその項目を無視する必要があるのかの理由を必ずコメントに記述してください。
  * ❌ `ignore_changes = all` の使用禁止。
  * `depends_on` は暗黙の依存関係が解決できない場合のみ、最小限に使用します。
* **Output 制限**:
  * 公開する Output は必要最小限とし、必ず `description` を定義してください。

---

## 8. CI/CD 前提ツール

生成・管理されるコードは、以下のツール群による自動検証が行われることを前提とします：
1. `terraform fmt` (フォーマット確認)
2. `terraform validate` (構文検証)
3. `tflint` (静的解析)
4. `tfsec` (セキュリティ診断)
5. `terraform plan` (差分確認)

---

## 9. AI 出力形式ルール

本スキルを適用して Terraform 関連の提案を行う際は、**コードスニペットの提示だけでなく、必ず以下の6点を含めて出力**してください。

1. **ディレクトリ構成**: 提案する環境やモジュールの物理的なツリー構造。
2. **State構成**: 各ディレクトリにおける State の分割方針と管理方法。
3. **Module責務**: 作成するモジュールがどのような役割（単一責任）を果たすか。
4. **Zone構成**: 提案インフラで使用するさくらクラウドのZone（東京/石狩）の設定。
5. **運用上の注意点**: 構築・運用の手順や、手動対応が必要な項目。
6. **想定リスク**: さくらクラウド特有の仕様（ディスクアタッチ限界やリソース上限など）によるリスクとその緩和策。
