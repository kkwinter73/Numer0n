package domain

// GenerateAllCandidates はあり得る全ての組み合わせ(720通り)を生成します。
// 0-9 の数字から重複なしで3桁を並べる順列の数 = 10P3 = 720。
func GenerateAllCandidates() []Secret {
	var c []Secret
	for i := 0; i <= 9; i++ {
		for j := 0; j <= 9; j++ {
			for k := 0; k <= 9; k++ {
				if i != j && j != k && i != k {
					c = append(c, Secret{i, j, k})
				}
			}
		}
	}
	return c
}

// FilterCandidates は guess と (eat, bite) の結果から
// 矛盾しない候補のみを残します。CPUの推論ロジックの中核。
func FilterCandidates(candidates []Secret, guess Secret, eat, bite int) []Secret {
	next := make([]Secret, 0, len(candidates))
	for _, c := range candidates {
		e, b := CheckEatBite(c, guess)
		if e == eat && b == bite {
			next = append(next, c)
		}
	}
	return next
}
