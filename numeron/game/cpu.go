package game

// GenerateAllCandidates はあり得る全ての組み合わせ(720通り)を生成します
func GenerateAllCandidates() [][]int {
	var c [][]int
	for i := 0; i <= 9; i++ {
		for j := 0; j <= 9; j++ {
			for k := 0; k <= 9; k++ {
				if i != j && j != k && i != k {
					c = append(c, []int{i, j, k})
				}
			}
		}
	}
	return c
}

// FilterCandidates は結果に基づき、あり得ない候補を除外します
func FilterCandidates(candidates [][]int, guess []int, eat, bite int) [][]int {
	var next [][]int
	for _, c := range candidates {
		testEat, testBite := CheckEatBite(c, guess)
		if testEat == eat && testBite == bite {
			next = append(next, c)
		}
	}
	return next
}
