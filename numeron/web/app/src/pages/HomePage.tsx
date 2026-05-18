import { Link } from "react-router-dom";

/**
 * Reactアプリのトップページ。
 *
 * 現状: 動作確認用のリンク集。
 * フェーズ4 以降でログイン画面・マイページ・ランキング等が追加される予定。
 *
 * 既存ゲーム本体 (CPU対戦・オンライン対戦) は web/static/index.html で動いており、
 * `/` でアクセスできる。このアプリは将来的にそれを置き換える前提だが、
 * フェーズ2.5 では「並列で動く別アプリ」の位置付け。
 */
export function HomePage() {
  return (
    <div className="page">
      <h1>NUMER0N — React App (足場)</h1>
      <p>
        ここは新規 React アプリの足場です。フェーズ2.5 で構築されました。
        既存のゲーム本体は <a href="/">こちら</a> (Goサーバー直配信)。
      </p>

      <nav>
        <ul>
          <li>
            <Link to="/app/health">Health Check (API疎通確認)</Link>
          </li>
        </ul>
      </nav>

      <p style={{ marginTop: "2rem", color: "#888", fontSize: "0.9em" }}>
        フェーズ4 (認証) 以降、ログイン・マイページ・ランキング等の画面がここに追加されます。
      </p>
    </div>
  );
}
