import { Route, Routes } from "react-router-dom";
import { HomePage } from "@/pages/HomePage";
import { HealthPage } from "@/pages/HealthPage";

/**
 * アプリケーションのルート定義。
 *
 * ルーティング戦略:
 *   - `/`         → 既存ゲーム本体 (web/static/index.html) が Go サーバーから配信される
 *                    (React アプリではアクセスされない想定。dev サーバーでは React のホームが出る)
 *   - `/app/*`    → React アプリの新規画面
 *
 * フェーズ6 で既存画面をReactに移行する際は、トップを React に切り替える。
 */
export function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/app" element={<HomePage />} />
      <Route path="/app/health" element={<HealthPage />} />
      <Route
        path="*"
        element={
          <div className="page">
            <h1>404 Not Found</h1>
            <p>ページが見つかりません。</p>
          </div>
        }
      />
    </Routes>
  );
}
