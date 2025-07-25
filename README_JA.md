# tosage

<p align="center">
  <img src="assets/icon.png" alt="tosage logo" width="256" height="256">
</p>

Claude CodeとCursorのトークン使用量を追跡し、Prometheusにメトリクスを送信するGoアプリケーションです。CLIモード（当日のトークン数を出力）またはデーモンモード（定期的にメトリクスを送信するシステムトレイアプリケーション）で実行できます。

## 機能

- **トークン使用量追跡**: Claude CodeとCursorの両方のトークン使用量を監視
- **Prometheus統合**: リモートライトAPIを介したメトリクス送信
- **デュアルモード動作**: 素早い確認のためのCLIモード、継続的な監視のためのデーモンモード
- **macOSシステムトレイ**: デーモンモード用のネイティブシステムトレイサポート
- **自動データ検出**: 複数の場所からClaude Codeのデータを自動検出
- **Cursor API統合**: プレミアムリクエストの使用状況と料金情報を取得

```mermaid
flowchart TD
    subgraph "データソース"
        CC[Claude Code<br/>ローカルディレクトリ]
        CA[Cursor API]
    end
    
    subgraph "インフラストラクチャ層"
        JSONL[JSONLリポジトリ<br/>Claude Codeデータファイル読み取り]
        CAPI[Cursor APIリポジトリ<br/>使用量データ取得]
        CDB[Cursor DBリポジトリ<br/>SQLiteストレージ]
    end
    
    subgraph "ユースケース層"
        CCS[Claude Codeサービス<br/>トークン使用量処理]
        CS[Cursorサービス<br/>APIデータ処理・トークン追跡]
        MS[メトリクスサービス<br/>メトリクス収集・集約]
    end
    
    subgraph "外部システム"
        PROM[Prometheus<br/>リモートライトAPI]
    end
    
    CC --> JSONL
    CA --> CAPI
    JSONL --> CCS
    CAPI --> CDB
    CDB --> CS
    CCS --> MS
    CS --> MS
    MS --> PROM
```

### Grafanaダッシュボード

事前設定済みのGrafanaダッシュボードでトークン使用量を監視:

![Grafana Dashboard](./assets/grafana.png)

[Grafanaダッシュボード設定](./assets/grafana.json)をインポートして以下を可視化:
- 1時間あたりの個人別トークン使用量
- チーム全体のトークン使用傾向
- 日別トークン合計
- ツール別内訳（Claude Code vs Cursor）

## インストール

### クイックインストール（推奨）

tosageをインストールする最も簡単な方法は、自動インストールスクリプトを使用することです：

```bash
# インストーラーをダウンロードして実行
curl -fsSL https://raw.githubusercontent.com/ca-srg/tosage/main/scripts/install.sh | bash
```

または、最初にスクリプトをダウンロードして確認する場合：

```bash
# スクリプトをダウンロード
curl -fsSL https://raw.githubusercontent.com/ca-srg/tosage/main/scripts/install.sh -o install-tosage.sh

# スクリプトを確認
less install-tosage.sh

# インストーラーを実行
bash install-tosage.sh
```

インストールスクリプトは以下を実行します：
1. お使いのアーキテクチャ（arm64/x86_64）に応じた最新のtosage DMGをダウンロード
2. `/Applications/tosage.app`にアプリケーションをインストール
3. 対話的に設定をガイド
4. `~/.config/tosage/config.json`に設定ファイルを作成

#### 前提条件

- macOS（インストーラーはmacOS専用）
- `curl`と`jq`（通常はプリインストール済み）
- GitHubからダウンロードするためのインターネット接続
- /Applicationsにインストールするための管理者パスワード

#### 必要な設定値

インストール中に以下の項目を入力します：
- **Prometheus リモートライトURL**（必須）：PrometheusエンドポイントURL（例：`https://prometheus.example.com/api/prom/push`）
- **Prometheus ユーザー名**（必須）：認証用ユーザー名
- **Prometheus パスワード**（必須）：認証用パスワード
- **ホストラベル**（任意）：メトリクス用のカスタムラベル
- **メトリクス間隔**（任意）：メトリクス送信間隔（秒）（デフォルト：600）
- **Promtail設定**（任意）：ログ転送用

### ビルド済みバイナリ

[GitHub Releases](https://github.com/ca-srg/tosage/releases)から最新リリースをダウンロードしてください。

### ソースからビルド

```bash
git clone https://github.com/ca-srg/tosage.git
cd tosage
make build
```

## 設定

```bash
# 1. アプリケーションを実行してconfig.jsonを生成

# 2. config.jsonを修正
$ cat ~/.config/tosage/config.json
{
  "prometheus": {
    "remote_write_url": "https://<prometheus_url>/api/prom/push",
    "username": "",
    "password": ""
  },
  "logging": {
    "promtail": {
      "url": "https://<logs_url>",
      "username": "",
      "password": ""
    }
  }
}

# 3. 再度実行
```

## 使用方法

### CLIモード

今日のトークン数を出力:

```bash
tosage
```

### デーモンモード

定期的にメトリクスを送信するシステムトレイアプリケーションとして実行:

```bash
tosage -d
```

## ビルド

### 必要要件

#### ビルド要件

- Go 1.21以上
- macOS（デーモンモード用）
- Make

#### 実行時要件

- メトリクス収集用のPrometheus Remote Write APIエンドポイント
- ログ集約用のGrafana Loki（オプション、Promtail経由）

### ビルドコマンド

```bash
# 現在のプラットフォーム用にビルド
make build

# macOS ARM64バイナリをビルド
make build-darwin

# macOS用アプリバンドルをビルド
make app-bundle-arm64

# DMGインストーラーをビルド
make dmg-arm64

# すべてのチェックを実行（fmt、vet、lint、test）
make check
```

### macOSアプリバンドルとDMG作成

#### アプリバンドルターゲット

##### `app-bundle-arm64`
**目的**: macOSアプリバンドル（.app）を作成

1. **バイナリビルド**: `build-darwin`を実行してGoバイナリを作成
2. **依存関係チェック**: `dmg-check`を実行して必要なツールを確認
3. **アプリバンドル作成**: `create-app-bundle.sh`を実行して以下を作成:
   - `tosage.app/Contents/MacOS/tosage` - 実行ファイル
   - `tosage.app/Contents/Info.plist` - アプリメタデータ
   - `tosage.app/Contents/Resources/app.icns` - アプリアイコン
   - `tosage.app/Contents/PkgInfo` - アプリタイプ情報

#### DMGターゲット

##### `dmg-arm64`
**目的**: 未署名のDMGインストーラーを作成

1. アプリバンドルを作成（`app-bundle-*`を実行）
2. `create-dmg.sh`を実行してDMGを作成:
   - DMGにアプリバンドルを含める
   - `/Applications`へのシンボリックリンクを追加
   - 背景画像とウィンドウレイアウトを設定
   - 出力: `tosage-{version}-darwin-{arch}.dmg`

##### `dmg-signed-arm64`
**目的**: 署名済みDMGを作成

- `CODESIGN_IDENTITY`環境変数が必要
- アプリバンドルとDMGにコード署名を追加

##### `dmg-notarized-arm64`
**目的**: 署名・公証済みDMGを作成

- 署名に加えてApple公証を追加
- Gatekeeperの警告なしでインストール可能

### ビルドプロセスフロー

```
Goソースコード
    ↓ (go build)
実行可能バイナリ
    ↓ (create-app-bundle.sh)
.appバンドル
    ↓ (create-dmg.sh)
.dmgインストーラー
    ↓ (codesign + 公証)
配布可能なDMG
```

### 使用例

#### 未署名DMGを作成:
```bash
make dmg-arm64
```

#### 署名済みDMGを作成:
```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
make dmg-signed-arm64
```

#### 署名・公証済みDMGを作成:
```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export API_KEY_ID="your-key-id"
export API_KEY_PATH="/path/to/AuthKey_XXXXX.p8"
export API_ISSUER="your-issuer-id"
make dmg-notarized-arm64
```

#### すべてのアーキテクチャ用に作成:
```bash
make dmg-notarized-all
```

## アーキテクチャ

本プロジェクトは関心の分離を明確にしたクリーンアーキテクチャに従っています：

### ドメイン層
- **エンティティ**: コアビジネスエンティティ（Claude Codeエントリ、Cursor使用データ）
- **リポジトリインターフェース**: データアクセスの抽象化
- **ドメインエラー**: ビジネスロジック固有のエラー

### インフラストラクチャ層
- **設定**: アプリケーション設定管理
- **依存性注入**: クリーンな依存関係管理のためのIoCコンテナ
- **ロギング**: 複数のロガー実装（debug、promtail）
- **リポジトリ実装**: 
  - 使用データ用Cursor APIクライアント
  - Cursorトークン履歴用SQLiteデータベース
  - Claude Codeデータ用JSONLリーダー
  - Prometheusリモートライトクライアント

### ユースケース層
- **サービス**: ビジネスロジック実装
  - Claude Codeデータ処理
  - Cursor API統合とトークン追跡
  - メトリクス収集と送信
  - アプリケーションステータス追跡

### インターフェース層
- **コントローラ**: アプリケーションエントリーポイント
  - コマンドラインインターフェース用CLIコントローラ
  - バックグラウンドサービス用デーモンコントローラ
  - UI用システムトレイコントローラ

## データソース

### Claude Code
以下の場所でデータを検索:
- `~/.config/claude/projects/`（新しいデフォルト）
- `~/.claude/projects/`（レガシー）
- `~/Library/Application Support/claude/projects/`（macOS）

### Cursor
Cursor APIを使用して以下を取得:
- プレミアム（GPT-4）リクエスト使用量
- 使用量ベースの料金情報
- チームメンバーシップステータス


## 注意事項

- macOSのみ（システムトレイにCGOを使用）
- 時刻計算はJST（アジア/東京）タイムゾーンを使用
- 設定ファイル: `~/.config/tosage/config.json`

## トラブルシューティング

### インストールの問題

#### "GitHub APIレート制限を超えました"
- レート制限がリセットされるまで待つ（通常1時間）
- または[GitHub Releases](https://github.com/ca-srg/tosage/releases)からDMGを手動でダウンロード

#### "DMGのマウントに失敗しました"
- 十分なディスク容量があることを確認
- 他のDMGがすでにマウントされていないか確認
- FinderからDMGを手動でマウントしてみる

#### インストール中の"アクセスが拒否されました"
- インストーラーは/Applicationsにコピーするため管理者権限が必要
- `sudo`でプロンプトが表示されたらパスワードを入力

#### "アーキテクチャ用のDMGが見つかりません"
- お使いのmacOSバージョンがサポートされているか確認
- Apple SiliconのMacの場合、arm64ビルドを使用していることを確認
- Intel Macの場合、x86_64ビルドはまだ利用できない可能性があります

### 設定の問題

#### "無効なURL形式"
- PrometheusのURLにプロトコル（http://またはhttps://）が含まれていることを確認
- 例：`https://prometheus.example.com/api/prom/push`

#### "設定ファイルのアクセスが拒否されました"
- 設定ファイルはセキュアな権限（600）で作成されます
- 手動で編集する必要がある場合は`sudo`を使用

### 実行時の問題

#### "tosage.appは壊れているため開くことができません"
- これはmacOS Gatekeeperの問題です
- インストーラーは自動的に検疫属性を削除します
- 問題が続く場合は次を実行：`sudo xattr -cr /Applications/tosage.app`

#### "Claude Codeのデータが見つかりません"
- Claude Codeが少なくとも一度実行されていることを確認
- データが検索ディレクトリのいずれかに存在することを確認
- 場所については「データソース」セクションを参照

## TODO

- [ ] Vertex AIトークン使用量追跡を追加
- [ ] Amazon Bedrockトークン使用量追跡を追加

## GitHub Actionsセットアップ

署名済みリリースをビルドしたいメンテナーは、必要な設定について[GitHub Secretsセットアップガイド](GITHUB_SECRETS_SETUP.md)を参照してください。

## ライセンス

MIT License