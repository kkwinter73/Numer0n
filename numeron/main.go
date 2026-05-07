package main

import (
	"fmt" // コンソールへのテキスト出力（Printfなど）を行うための標準パッケージ
	"log"
	"net/http"
	"numeron/handler" // ブラウザからのリクエストを処理する「プレゼンテーション層」の独自パッケージ
	"numeron/store"   // ゲームの進行状態を保存する「インフラ・データ層」の独自パッケージ
)

func main() {
	// ==========================================
	// 1. 依存関係のセットアップ（準備フェーズ）
	// ==========================================

	// メモリ上でゲームのセッション（進行状態）を管理するための保存場所を作成
	sessionStore := store.NewSessionStore()

	// ブラウザからのリクエストを受け付ける「窓口担当（ハンドラー）」を作成します。
	// 上で作った保存場所を渡して窓口担当がデータを読み書きできるようにする。
	apiHandler := handler.NewAPIHandler(sessionStore)

	// ==========================================
	// 2. ルーティングの設定（URLの交通整理）
	// ==========================================

	// URLのルート（ http://localhost:8080/ ）にアクセスが来たら、
	// "static" フォルダの中身（index.htmlなど）をそのままブラウザに返して画面を表示させます。
	http.Handle("/", http.FileServer(http.Dir("static")))

	// ブラウザから "ゲーム開始" のリクエスト（/api/start）が来たら、
	// apiHandler の HandleStart 関数を実行して処理させます。
	http.HandleFunc("/api/start", apiHandler.HandleStart)

	// ブラウザから "数字の予想" のリクエスト（/api/guess）が来たら、
	// apiHandler の HandleGuess 関数を実行して処理させます。
	http.HandleFunc("/api/guess", apiHandler.HandleGuess)

	// ==========================================
	// 3. サーバーの起動
	// ==========================================

	// サーバーが待ち受けるポート番号（8080番）を変数に設定します。
	port := ":8080"

	// ターミナル（黒い画面）に、サーバーが起動したこととアクセス先のURLを分かりやすく表示します。
	fmt.Printf("🚀 サーバーを起動しました: http://localhost%s\n", port)

	// http.ListenAndServe で実際にWebサーバーを起動し、リクエストの待ち受けを開始します。
	// ずっと動き続けますが、もしポートが既に使われているなどの深刻なエラーで起動に失敗した場合は、
	// 外側を囲っている log.Fatal() がエラー内容を出力し、プログラムを安全に強制終了させます。
	log.Fatal(http.ListenAndServe(port, nil))
}
