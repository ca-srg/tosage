# macOS Gatekeeper Guide for tosage

## 概要
tosageはオープンソースのアプリケーションで、Apple Developer IDで署名されていません。そのため、初回実行時にmacOSのセキュリティ機能（Gatekeeper）により「開発元が未確認」または「壊れている」というエラーが表示される場合があります。

これは**正常な動作**であり、アプリケーション自体に問題はありません。

## 実行方法

### 方法1: 右クリックで開く（推奨）

1. **DMGファイルを開く**
   - ダウンロードした`.dmg.zip`ファイルを展開
   - DMGファイルをダブルクリックしてマウント

2. **アプリケーションをコピー**
   - `tosage.app`をApplicationsフォルダにドラッグ＆ドロップ

3. **初回起動**
   - Finderで`/Applications`を開く
   - `tosage.app`を**右クリック**（またはControl+クリック）
   - メニューから「開く」を選択
   - 警告ダイアログで「開く」ボタンをクリック

4. **以降の起動**
   - 一度この手順で開いた後は、通常通りダブルクリックで起動可能

### 方法2: ターミナルから実行

```bash
# アプリケーションの実行権限を確認
xattr -l /Applications/tosage.app

# 検疫属性を削除
sudo xattr -cr /Applications/tosage.app

# 実行
/Applications/tosage.app/Contents/MacOS/tosage
```

### 方法3: システム設定から許可

1. アプリケーションをダブルクリックして起動を試みる
2. 「"tosage"は開発元が未確認のため開けません」というエラーが表示される
3. **システム設定** > **プライバシーとセキュリティ**を開く
4. 「セキュリティ」セクションに「"tosage"は開発元が未確認のため...」というメッセージが表示される
5. 「このまま開く」ボタンをクリック

## よくある質問

### Q: なぜこのようなエラーが表示されるのですか？
A: macOSは、Apple Developer IDで署名されていないアプリケーションの実行を制限しています。これはマルウェアからユーザーを保護するためのセキュリティ機能です。

### Q: アプリケーションは安全ですか？
A: tosageはオープンソースソフトウェアであり、ソースコードは[GitHub](https://github.com/ca-srg/tosage)で公開されています。誰でもコードを確認できます。

### Q: 毎回右クリックする必要がありますか？
A: いいえ、初回のみです。一度「開く」を選択してアプリケーションを実行すると、以降は通常通りダブルクリックで起動できます。

### Q: 「壊れているため開けません」と表示される場合は？
A: 以下のコマンドを実行してください：
```bash
# 修正スクリプトを実行
./scripts/fix-app-signature.sh /Applications/tosage.app

# または手動で属性を削除
sudo xattr -cr /Applications/tosage.app
sudo codesign --force --sign - /Applications/tosage.app
```

## 技術的詳細

### Gatekeeperとは
Gatekeeperは、インターネットからダウンロードしたアプリケーションの実行を制御するmacOSのセキュリティ機能です。

### 検疫属性（Quarantine Attribute）
macOSは、インターネットからダウンロードしたファイルに`com.apple.quarantine`という拡張属性を付与します。この属性があると、Gatekeeperがファイルの安全性をチェックします。

### Developer ID署名
Apple Developer Programに登録した開発者は、Developer ID証明書でアプリケーションに署名できます。署名されたアプリケーションは、Gatekeeperのチェックをパスしやすくなります。

## 参考リンク
- [Apple Support - Gatekeeper](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unidentified-developer-mh40616/mac)
- [tosage GitHub Repository](https://github.com/ca-srg/tosage)