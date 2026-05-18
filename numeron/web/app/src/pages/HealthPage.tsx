import { useQuery } from "@tanstack/react-query";
import { numeronApi } from "@/api/numeron";

/**
 * 動作確認用ページ: ヘルスチェック API を叩いて結果表示。
 *
 * 確認できること:
 *   - React アプリのビルド・起動
 *   - Vite proxy 経由でGo サーバーに通信できる
 *   - TanStack Query のセットアップが正しい
 *   - 型推論が効いている (data の型が HealthResponse)
 */
export function HealthPage() {
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["health"],
    queryFn: () => numeronApi.health(),
  });

  return (
    <div className="page">
      <h1>Health Check</h1>
      <p>Goサーバーの /api/health を呼び出して結果を表示します。</p>

      <div className="card">
        {isLoading && <p>Loading…</p>}
        {error && (
          <p className="error">
            Error: {error.message}
          </p>
        )}
        {data && (
          <>
            <p>
              Status: <strong>{data.status}</strong>
            </p>
            {data.dependencies && data.dependencies.length > 0 && (
              <ul>
                {data.dependencies.map((dep) => (
                  <li key={dep.name}>
                    {dep.name}: {dep.status}
                    {dep.error && ` (${dep.error})`}
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </div>

      <button onClick={() => refetch()} disabled={isFetching}>
        {isFetching ? "Refetching…" : "Refetch"}
      </button>
    </div>
  );
}
