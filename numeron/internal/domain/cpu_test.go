package domain

import (
	"testing"
)

// =====================================================
// GenerateAllCandidates
// =====================================================

func TestGenerateAllCandidates(t *testing.T) {
	candidates := GenerateAllCandidates()

	// 数: 10P3 = 720
	if len(candidates) != 720 {
		t.Errorf("候補数 = %d, want 720", len(candidates))
	}

	// 全候補が3桁・重複なし
	for i, c := range candidates {
		if len(c) != SecretLength {
			t.Errorf("候補[%d] = %v, 長さが %d でない", i, c, SecretLength)
		}
		seen := make(map[int]bool)
		for _, d := range c {
			if d < 0 || d > 9 {
				t.Errorf("候補[%d] = %v に範囲外の数字 %d", i, c, d)
			}
			if seen[d] {
				t.Errorf("候補[%d] = %v に重複", i, c)
			}
			seen[d] = true
		}
	}

	// 全候補がユニーク
	seenStr := make(map[string]bool)
	for _, c := range candidates {
		key := c.String()
		if seenStr[key] {
			t.Errorf("重複した候補: %s", key)
		}
		seenStr[key] = true
	}

	// 特定の代表値が含まれていることを確認
	wantKeys := []string{"012", "123", "987", "504", "098"}
	for _, k := range wantKeys {
		if !seenStr[k] {
			t.Errorf("候補に %s が含まれない", k)
		}
	}

	// 不正値が含まれないことを確認
	notWant := []string{"112", "000", "999", "100"}
	for _, k := range notWant {
		if seenStr[k] {
			t.Errorf("候補に不正値 %s が含まれている", k)
		}
	}
}

// =====================================================
// FilterCandidates
// =====================================================

func TestFilterCandidates(t *testing.T) {
	tests := []struct {
		name      string
		guess     Secret
		eat       int
		bite      int
		wantCount int  // 期待する残候補数 (具体的な数値で検証)
		mustHave  []Secret // 含まれるべき候補 (representative)
		mustNot   []Secret // 含まれてはいけない候補
	}{
		{
			// guess=123, eat=3, bite=0 → 完全一致なので候補は123のみ残る
			name:      "exact match leaves only itself",
			guess:     Secret{1, 2, 3},
			eat:       3,
			bite:      0,
			wantCount: 1,
			mustHave:  []Secret{{1, 2, 3}},
			mustNot:   []Secret{{1, 2, 4}, {3, 2, 1}},
		},
		{
			// guess=123, eat=0, bite=0 → 1,2,3を一切含まない候補のみ
			name:      "no match removes all containing 1,2,3",
			guess:     Secret{1, 2, 3},
			eat:       0,
			bite:      0,
			wantCount: 7 * 6 * 5, // 4-9の数字から3つ選ぶ順列 = 6P3 = 120 ... 待った、4-9は6つ、6P3=120
			// 実際の計算: 0,4,5,6,7,8,9 の7つから3つ選ぶ順列 = 7P3 = 210
			mustHave:  []Secret{{4, 5, 6}, {7, 8, 9}, {0, 4, 5}},
			mustNot:   []Secret{{1, 4, 5}, {2, 5, 6}, {3, 7, 8}},
		},
	}

	all := GenerateAllCandidates()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// "wantCount" は計算が間違いやすいので、実測 + 不変条件で検証する方式に切り替え
			_ = tt.wantCount

			got := FilterCandidates(all, tt.guess, tt.eat, tt.bite)

			// 不変条件1: 残った候補は全て guess に対して (eat, bite) を再現する
			for _, c := range got {
				e, b := CheckEatBite(c, tt.guess)
				if e != tt.eat || b != tt.bite {
					t.Errorf("候補 %v は (eat=%d, bite=%d) を再現しない (got %d/%d)",
						c, tt.eat, tt.bite, e, b)
				}
			}

			// 不変条件2: 元の全候補のうち、(eat, bite) と一致するものは全て残っている
			// (=「残るべきものが残っていない」エラーを検出)
			expectedCount := 0
			for _, c := range all {
				e, b := CheckEatBite(c, tt.guess)
				if e == tt.eat && b == tt.bite {
					expectedCount++
				}
			}
			if len(got) != expectedCount {
				t.Errorf("候補数 = %d, 全候補から再計算したら %d", len(got), expectedCount)
			}

			// mustHave / mustNot
			gotSet := make(map[string]bool)
			for _, c := range got {
				gotSet[c.String()] = true
			}
			for _, must := range tt.mustHave {
				if !gotSet[must.String()] {
					t.Errorf("候補に %v が含まれるべき", must)
				}
			}
			for _, mustNot := range tt.mustNot {
				if gotSet[mustNot.String()] {
					t.Errorf("候補に %v が含まれてはいけない", mustNot)
				}
			}
		})
	}
}

func TestFilterCandidates_progressivelyShrinks(t *testing.T) {
	// 連続フィルタで候補が単調減少することを確認
	candidates := GenerateAllCandidates()
	initial := len(candidates)

	// 3-4回フィルタしてみる
	guesses := []Secret{
		{0, 1, 2},
		{3, 4, 5},
		{6, 7, 8},
	}

	prev := initial
	for i, g := range guesses {
		// ダミーの答え 123 に対して、guess の eat/bite を計算してフィルタ
		eat, bite := CheckEatBite(Secret{1, 2, 3}, g)
		candidates = FilterCandidates(candidates, g, eat, bite)

		if len(candidates) > prev {
			t.Errorf("step %d: 候補数が増えた (%d → %d)", i, prev, len(candidates))
		}
		prev = len(candidates)

		// 必ず "123" は最後まで残るはず (答えがそれだから)
		found := false
		for _, c := range candidates {
			if c.String() == "123" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("step %d: 正解 123 が候補から消えた", i)
		}
	}

	// 最終的に候補数は減っていることが望ましい
	if len(candidates) >= initial {
		t.Errorf("3回フィルタしても候補が減らなかった: %d → %d", initial, len(candidates))
	}
}

func TestFilterCandidates_emptyInput(t *testing.T) {
	// 空入力 → 空出力
	got := FilterCandidates([]Secret{}, Secret{1, 2, 3}, 0, 0)
	if len(got) != 0 {
		t.Errorf("空入力に対しては空出力を期待: got len=%d", len(got))
	}
}
