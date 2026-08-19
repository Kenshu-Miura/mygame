# UFO撃ち落としたことありますか？

[Ebitengine](https://ebitengine.org/) で作られた、ブラウザとデスクトップで遊べる2Dシューティングゲームです。

## 操作方法

| キー | 操作 |
| --- | --- |
| `Space` | タイトル画面でゲーム開始 / ゲーム中に弾を発射（長押しで連射） |
| `←` `→` | プレイヤーを左右に移動 |
| `↑` | KIEE Countを20消費して画面上の敵を一掃 |
| `Esc` | タイトル画面へ戻ってリスタート |

- UFOやボスへ連続で弾を当てるとコンボが増え、5コンボごとにスコア倍率が上がります（最大5倍）。
- エビに弾を当てるとスコアが2減ります（最低0点）。
- エビへの誤射、または敵への接触でコンボが0に戻ります。
- 弾が画面外へ抜けると KIEE Count が1増えます。
- UFOを倒すと20%の確率で「P」アイテムが落下します。取得すると10秒間、上・左斜め上・右斜め上へ広がる3WAYショットになります。未取得のアイテムと発動中の効果はウェーブをまたいで残ります。
- KIEE Countは画面左上のゲージで確認でき、20まで溜まるとゲージが金色になります。
- ハイスコアは自動保存され、タイトル画面とゲーム画面に表示されます。
- 上から落ちてくる敵に触れるとゲームオーバーです。
- UFOと通常のエビは画面の左右どちらからも出現し、ウェーブが進むほど横移動が速くなります。上から落ちる敵もウェーブごとに速くなります。
- 通常ウェーブは指定数のUFOを倒すとクリアです。必要数は「4 + 現在のウェーブ」（ウェーブ1は5体、最大20体）で、画面左上に `UFO: 撃破数/必要数` と表示されます。
- 5ウェーブごとに巨大海老ボスが出現し、ボスウェーブはボスのHPを0にするとクリアです。
- ボスはランダムなタイミングで左右の移動方向を変えながら攻撃し、HPを0にするとボーナススコアを獲得して次のウェーブへ進みます。

## 必要な環境

- Go 1.25.13 以上
- Ebitengine v2.9.10（`go.mod` で固定）
- Web版はPowerShellまたはBashでビルド可能

## デスクトップ版を実行する

```sh
go mod download
go run .
```

## WebAssembly版をビルドする

```sh
bash scripts/build-web.sh
```

Windows PowerShellでは次を実行します。

```powershell
./scripts/build-web.ps1
```

`dist/` に次の公開用ファイルが生成されます。

- `index.html`
- `game.html`
- `mygame.wasm`
- ビルドに使用した Go と同じバージョンの `wasm_exec.js`
- 画像・音声アセット

ブラウザ版のゲーム画面は4:3の比率を保ち、端末の画面サイズに合わせて最大960px幅まで拡大されます。

ブラウザのセキュリティ制約により `index.html` を直接開くことはできません。ローカルHTTPサーバーで確認してください。

```sh
python -m http.server 8080 --directory dist
```

その後、<http://localhost:8080> を開きます。

## デバッグモード

ブラウザ版はURLの末尾に `?debug=1` を付けるとデバッグモードになります。

```text
http://localhost:8080/?debug=1
```

デスクトップ版は環境変数を指定して起動します。

```powershell
$env:MYGAME_DEBUG = "1"
go run .
```

デバッグモードでは落下する敵に触れてもゲームオーバーになりません。衝突はログへ記録され、接触した敵だけが取り除かれます。ブラウザ版のログは開発者ツールのConsole、デスクトップ版のログは起動したターミナルで確認できます。

| キー | デバッグ操作 |
| --- | --- |
| `B` | 現在または次のボスウェーブへ移動 |
| `K` | KIEE Countを必殺技が使える20まで補充 |
| `P` | プレイヤーの上にパワーアップアイテムを出現させる |

通常モードで敵に触れてゲームオーバーになった場合も、ウェーブ、スコア、コンボ、プレイヤーと敵の座標がログへ記録されます。

## ハイスコアの保存場所

- ブラウザ版: 公開サイトのオリジンごとにブラウザの `localStorage` へ保存
- デスクトップ版: OSのユーザー設定フォルダ内の `mygame/highscore` へ保存

ブラウザのサイトデータを削除した場合や、別のドメインでゲームを開いた場合は別のハイスコアとして扱われます。

## Netlifyで公開する

このリポジトリには [`netlify.toml`](netlify.toml) と [`.go-version`](.go-version) が含まれています。Netlifyで追加のビルド設定を入力する必要はありません。

1. このリポジトリをGitHubへpushします。
2. Netlifyの **Add new project** → **Import an existing project** を選びます。
3. GitHubを選択し、このリポジトリを指定します。
4. 設定値が次の内容になっていることを確認してデプロイします。

   - Build command: `bash scripts/build-web.sh`
   - Publish directory: `dist`
   - Go version: `.go-version` の `1.25.13`

以後はNetlifyで設定したProduction branch（通常は `main`）へpushするたびに、WASMが再ビルドされて自動公開されます。

Netlify CLIを使う場合は、リポジトリのルートで次を実行します。

```sh
netlify build
netlify deploy
netlify deploy --prod
```

最初の `netlify deploy` でサイトの選択または新規作成を求められます。本番反映前にDraft DeployのURLで動作確認してください。

## プロジェクト構成

```text
.
├── main.go               # ゲーム本体
├── main_test.go          # 連射・コンボ・ボス・パワーアップのテスト
├── highscore_*.go        # ブラウザ・デスクトップ別のハイスコア保存
├── boss_ebi.png          # 巨大海老ボスの透過ドット絵
├── web/                  # Webページとゲームiframeのソース
├── scripts/build-web.sh  # Netlify / Bash用Webビルド
├── scripts/build-web.ps1 # Windows PowerShell用Webビルド
├── netlify.toml          # Netlify設定
├── .go-version           # Netlifyで使用するGoバージョン
└── dist/                 # 生成物（Git管理対象外）
```

## ライセンス・素材

ゲーム内の画像・音声素材を再利用する場合は、各素材の権利を確認してください。Ebitengine本体はApache License 2.0です。
`boss_ebi.png` は、このゲームのボス素材としてOpenAIの画像生成機能で作成した透過ドット絵です。
