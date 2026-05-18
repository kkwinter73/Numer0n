import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Vite 設定
// docs: https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],

  // パスエイリアス: src/foo → '@/foo'
  // tsconfig.json の paths と同期する
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },

  // 開発サーバー
  server: {
    port: 5173,
    strictPort: true,
    // /api と /ws へのリクエストは Go サーバー (localhost:8080) に転送する。
    // これにより、開発時はフロントを 5173、バックを 8080 で同時起動できる。
    // CORS / Cookie の心配がなくなる。
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: false,
      },
      // フェーズ3 で WebSocket を追加する際に使う
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        changeOrigin: false,
      },
    },
  },

  // 本番ビルド
  build: {
    // 出力先は web/app/dist/
    // 将来 Cloudflare Pages にデプロイする想定。Goサーバー側で配信するなら
    // web/static/ に出力先を変える運用も可。
    outDir: "dist",
    sourcemap: true,
  },
});
