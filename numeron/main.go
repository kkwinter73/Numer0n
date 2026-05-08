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
	roomStore := store.NewRoomStore()

	// ブラウザからのリクエストを受け付ける「窓口担当（ハンドラー）」を作成します。
	// 上で作った保存場所を渡して窓口担当がデータを読み書きできるようにする。
	apiHandler := handler.NewAPIHandler(sessionStore)
	onlineHandler := handler.NewOnlineHandler(roomStore)

	// ==========================================
	// 2. ルーティングの設定（URLの交通整理）
	// ==========================================

	// 静的ファイル
	http.Handle("/", http.FileServer(http.Dir("static")))

	// CPU対戦
	http.HandleFunc("/api/start", apiHandler.HandleStart)
	http.HandleFunc("/api/guess", apiHandler.HandleGuess)

	// オンライン対戦
	http.HandleFunc("/api/online/create", onlineHandler.HandleCreate)
	http.HandleFunc("/api/online/join", onlineHandler.HandleJoin)
	http.HandleFunc("/api/online/state", onlineHandler.HandleState)
	http.HandleFunc("/api/online/secret", onlineHandler.HandleSecret)
	http.HandleFunc("/api/online/guess", onlineHandler.HandleGuess)
	http.HandleFunc("/api/online/poll", onlineHandler.HandlePoll)

	// ==========================================
	// 3. サーバーの起動
	// ==========================================

	port := ":8080"
	fmt.Printf("🚀 サーバーを起動しました: http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
