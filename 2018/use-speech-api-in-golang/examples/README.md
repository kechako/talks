# Google Cloud Speech API Go example

## 認証

* [Google Cloud Console][cloud-console] で新しいプロジェクトを作成し、[Speech API][speech-api] を有効にします。
* Cloud Console にてサービスアカウントを作成し、サービスアカウントの認証情報 JSON ファイルをダウンロードし、パスを 
  `GOOGLE_APPLICATION_CREDENTIALS` にセットします:

  ```bash
  export GOOGLE_APPLICATION_CREDENTIALS=/path/to/your-project-credentials.json
  ```

[cloud-console]: https://console.cloud.google.com
[speech-api]: https://console.cloud.google.com/apis/api/speech.googleapis.com/overview?project=_

## サンプルの実行

マイク入力のために PortAudio を使用しているので、実行するプラットフォームに合わせて PortAudio をインストールします。

例えば macOS で Homebrew を使用している場合は:

```bash
brew install portaudio
```

サンプルを実行するまえに、まず依存するパッケージをインストールする必要があります:

```bash
go get -u cloud.google.com/go/speech/apiv1
go get -u github.com/gordonklaus/portaudio
```

非同期音声認識のサンプルを実行するには:

```bash
go run recognize.go
```

ストリーミング音声認識のサンプルを実行するには:

```bash
go run livecaption.go
```

