// Package domain はNumeronのドメインモデルとビジネスルールを定義します。
// このパッケージは外部の依存(DB、HTTP、フレームワーク等)を一切持たず、
// 純粋なゲームロジックのみを表現します。
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"strconv"
)

// SecretLength は暗証番号の桁数です。Numeronの標準ルールに従い3桁。
const SecretLength = 3

// Secret は3桁・重複なしの暗証番号を表すドメインモデルです。
// 内部表現は []int ですが、不正な値を持てないように
// ParseSecret 経由でのみ生成されます。
type Secret []int

// ParseSecret は文字列を検証してSecretに変換します。
// 3桁、すべて数字、重複なし、を満たさない場合はエラーを返します。
func ParseSecret(input string) (Secret, error) {
	if len(input) != SecretLength {
		return nil, fmt.Errorf("%d桁で入力してください", SecretLength)
	}
	out := make(Secret, SecretLength)
	seen := make(map[int]bool)
	for i, ch := range input {
		n, err := strconv.Atoi(string(ch))
		if err != nil {
			return nil, fmt.Errorf("数字のみ入力可能です")
		}
		if seen[n] {
			return nil, fmt.Errorf("重複した数字は使えません")
		}
		seen[n] = true
		out[i] = n
	}
	return out, nil
}

// String は "123" のような文字列表現を返します。
func (s Secret) String() string {
	if len(s) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(s))
	for _, d := range s {
		buf = append(buf, byte('0'+d))
	}
	return string(buf)
}

// CheckEatBite は target と guess を比較し、EAT(位置・数字一致)と BITE(数字のみ一致)を返します。
// Numeron の中核ルール。純粋関数。
func CheckEatBite(target, guess Secret) (eat, bite int) {
	for i := 0; i < SecretLength; i++ {
		for j := 0; j < SecretLength; j++ {
			if guess[i] == target[j] {
				if i == j {
					eat++
				} else {
					bite++
				}
			}
		}
	}
	return eat, bite
}

// GenerateRandomSecret は重複しない3桁の Secret を生成します。
// math/rand を使うため、暗号学的乱数ではないことに注意。
func GenerateRandomSecret() Secret {
	digits := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	mrand.Shuffle(len(digits), func(i, j int) {
		digits[i], digits[j] = digits[j], digits[i]
	})
	return Secret(digits[:SecretLength])
}

// GenerateID は対戦やセッション用の一意なIDを生成します。
// crypto/rand を使う暗号学的乱数。
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
