package domain

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// =====================================================
// ParseSecret
// =====================================================

func TestParseSecret(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Secret
		wantErr   bool
		errSubstr string // エラー時、メッセージにこの文字列が含まれていること
	}{
		// 正常系
		{name: "valid: 123", input: "123", want: Secret{1, 2, 3}},
		{name: "valid: 012 (leading zero)", input: "012", want: Secret{0, 1, 2}},
		{name: "valid: 987", input: "987", want: Secret{9, 8, 7}},
		{name: "valid: 504", input: "504", want: Secret{5, 0, 4}},

		// 桁数エラー
		{name: "too short: 12", input: "12", wantErr: true, errSubstr: "3桁"},
		{name: "too long: 1234", input: "1234", wantErr: true, errSubstr: "3桁"},
		{name: "empty string", input: "", wantErr: true, errSubstr: "3桁"},

		// 非数字エラー
		{name: "letters: abc", input: "abc", wantErr: true, errSubstr: "数字"},
		{name: "mixed: 1a2", input: "1a2", wantErr: true, errSubstr: "数字"},
		{name: "symbol: 1-2", input: "1-2", wantErr: true, errSubstr: "数字"},
		{name: "space: 1 2", input: "1 2", wantErr: true, errSubstr: "数字"},

		// 重複エラー
		{name: "duplicate: 112", input: "112", wantErr: true, errSubstr: "重複"},
		{name: "duplicate: 121", input: "121", wantErr: true, errSubstr: "重複"},
		{name: "all same: 111", input: "111", wantErr: true, errSubstr: "重複"},
		{name: "duplicate: 988", input: "988", wantErr: true, errSubstr: "重複"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSecret(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSecret(%q): エラーが返ることを期待したが nil だった", tt.input)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseSecret(%q): エラーメッセージ %q は %q を含まない",
						tt.input, err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseSecret(%q): 予期せぬエラー: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSecret(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// =====================================================
// Secret.String
// =====================================================

func TestSecret_String(t *testing.T) {
	tests := []struct {
		name string
		s    Secret
		want string
	}{
		{name: "normal", s: Secret{1, 2, 3}, want: "123"},
		{name: "with zero", s: Secret{0, 1, 2}, want: "012"},
		{name: "trailing zero", s: Secret{5, 0, 4}, want: "504"},
		{name: "all nines", s: Secret{9, 9, 9}, want: "999"}, // 不正値だがStringは動く
		{name: "empty", s: Secret{}, want: ""},
		{name: "nil", s: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Secret%v.String() = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

// =====================================================
// CheckEatBite
// =====================================================

func TestCheckEatBite(t *testing.T) {
	tests := []struct {
		name     string
		target   Secret
		guess    Secret
		wantEat  int
		wantBite int
	}{
		// 完全一致 (3 EAT, 0 BITE)
		{name: "exact match: 123 vs 123", target: Secret{1, 2, 3}, guess: Secret{1, 2, 3}, wantEat: 3, wantBite: 0},
		{name: "exact match: 087 vs 087", target: Secret{0, 8, 7}, guess: Secret{0, 8, 7}, wantEat: 3, wantBite: 0},

		// 完全すれ違い (0 EAT, 3 BITE - 全数字一致だが位置全て違い)
		{name: "all bite: 123 vs 312", target: Secret{1, 2, 3}, guess: Secret{3, 1, 2}, wantEat: 0, wantBite: 3},
		{name: "all bite: 123 vs 231", target: Secret{1, 2, 3}, guess: Secret{2, 3, 1}, wantEat: 0, wantBite: 3},

		// 部分一致
		{name: "1 eat 1 bite: 123 vs 132", target: Secret{1, 2, 3}, guess: Secret{1, 3, 2}, wantEat: 1, wantBite: 2},
		{name: "2 eat 0 bite: 123 vs 124", target: Secret{1, 2, 3}, guess: Secret{1, 2, 4}, wantEat: 2, wantBite: 0},
		{name: "1 eat 0 bite: 123 vs 145", target: Secret{1, 2, 3}, guess: Secret{1, 4, 5}, wantEat: 1, wantBite: 0},
		{name: "0 eat 1 bite: 123 vs 456 contains 1...wait", target: Secret{1, 2, 3}, guess: Secret{4, 5, 1}, wantEat: 0, wantBite: 1},
		{name: "0 eat 2 bite: 123 vs 213", target: Secret{1, 2, 3}, guess: Secret{2, 1, 4}, wantEat: 0, wantBite: 2},

		// 完全不一致 (0 EAT, 0 BITE)
		{name: "no match: 123 vs 456", target: Secret{1, 2, 3}, guess: Secret{4, 5, 6}, wantEat: 0, wantBite: 0},
		{name: "no match: 012 vs 789", target: Secret{0, 1, 2}, guess: Secret{7, 8, 9}, wantEat: 0, wantBite: 0},

		// 0が絡むケース
		{name: "with zero: 012 vs 021", target: Secret{0, 1, 2}, guess: Secret{0, 2, 1}, wantEat: 1, wantBite: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eat, bite := CheckEatBite(tt.target, tt.guess)
			if eat != tt.wantEat || bite != tt.wantBite {
				t.Errorf("CheckEatBite(%v, %v) = (%d, %d), want (%d, %d)",
					tt.target, tt.guess, eat, bite, tt.wantEat, tt.wantBite)
			}

			// 不変条件: eat + bite <= 3 (3桁・重複なし前提)
			if eat+bite > SecretLength {
				t.Errorf("eat(%d) + bite(%d) > %d: 3桁・重複なし前提では不可能", eat, bite, SecretLength)
			}

			// 不変条件: eat == 3 なら bite == 0
			if eat == 3 && bite != 0 {
				t.Errorf("eat==3 のとき bite は 0 でなければならないが %d だった", bite)
			}
		})
	}
}

// =====================================================
// GenerateRandomSecret
// =====================================================

func TestGenerateRandomSecret(t *testing.T) {
	// 確率的なテストなので、多数回呼び出して不変条件を確認
	const iterations = 1000

	for i := 0; i < iterations; i++ {
		s := GenerateRandomSecret()

		// 桁数
		if len(s) != SecretLength {
			t.Fatalf("iter %d: len = %d, want %d", i, len(s), SecretLength)
		}

		// 各桁が 0-9 の範囲
		for j, d := range s {
			if d < 0 || d > 9 {
				t.Fatalf("iter %d: index %d = %d, out of [0,9]", i, j, d)
			}
		}

		// 重複なし
		seen := make(map[int]bool)
		for _, d := range s {
			if seen[d] {
				t.Fatalf("iter %d: 重複 %d in %v", i, d, s)
			}
			seen[d] = true
		}
	}
}

func TestGenerateRandomSecret_distribution(t *testing.T) {
	// 各桁の出現が極端に偏らないことを確認 (簡易チェック)
	// 10000回呼んで、各数字 0-9 が最初の桁に現れる回数を数える
	const iterations = 10000
	counts := make(map[int]int)

	for i := 0; i < iterations; i++ {
		s := GenerateRandomSecret()
		counts[s[0]]++
	}

	// 期待値は iterations / 10 = 1000。±50% 以内に収まることを確認
	expected := iterations / 10
	minAllowed := expected / 2
	maxAllowed := expected + expected/2
	for d, c := range counts {
		if c < minAllowed || c > maxAllowed {
			t.Errorf("数字 %d の出現回数 %d が想定範囲 [%d, %d] を外れた",
				d, c, minAllowed, maxAllowed)
		}
	}
}

// =====================================================
// GenerateID
// =====================================================

func TestGenerateID(t *testing.T) {
	// 形式チェック (hex 16文字 = 8バイト)
	hexRe := regexp.MustCompile(`^[0-9a-f]{16}$`)

	id := GenerateID()
	if !hexRe.MatchString(id) {
		t.Errorf("GenerateID() = %q, want 16桁の16進文字列", id)
	}

	// 一意性チェック (重複の確率は無視できるほど低い)
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateID()
		if seen[id] {
			t.Fatalf("重複したID生成: %s", id)
		}
		seen[id] = true
	}
}
