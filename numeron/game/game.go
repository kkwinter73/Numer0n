package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand" // 「mrand」というエイリアスを付ける
	"strconv"
	"time"
)

// Session は1つのゲーム状態を表すドメインモデルです
type Session struct {
	ID            string    `json:"id"`
	PlayerSecret  []int     `json:"-"`
	CpuSecret     []int     `json:"-"`
	CpuCandidates [][]int   `json:"-"`
	Turn          int       `json:"turn"`
	Status        string    `json:"status"` // "playing", "player_win", "cpu_win", "draw"
	Logs          []TurnLog `json:"logs"`
	// 試合終了時のみクライアントに開示する数字 (omitempty なのでプレイ中は出ない)
	RevealedCpu string `json:"revealed_cpu,omitempty"`
	RevealedYou string `json:"revealed_you,omitempty"`
}

// FormatSecret は数字スライスを"123"形式の文字列に変換します
func FormatSecret(s []int) string {
	if len(s) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(s))
	for _, d := range s {
		buf = append(buf, byte('0'+d))
	}
	return string(buf)
}

type TurnLog struct {
	Turn        int    `json:"turn"`
	PlayerGuess string `json:"player_guess"`
	PlayerEat   int    `json:"player_eat"`
	PlayerBite  int    `json:"player_bite"`
	CpuGuess    string `json:"cpu_guess"`
	CpuEat      int    `json:"cpu_eat"`
	CpuBite     int    `json:"cpu_bite"`
}

func init() {
	// math/rand の方は mrand として呼び出す
	mrand.Seed(time.Now().UnixNano())
}

// GenerateID は一意のセッションIDを生成します
func GenerateID() string {
	b := make([]byte, 8)
	// こちらは crypto/rand なのでそのまま rand.Read を使う
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseInput は入力文字列を検証し、スライスに変換します
func ParseInput(input string) ([]int, error) {
	if len(input) != 3 {
		return nil, fmt.Errorf("3桁で入力してください")
	}
	guess := make([]int, 3)
	seen := make(map[int]bool)
	for i, char := range input {
		num, err := strconv.Atoi(string(char))
		if err != nil || seen[num] {
			return nil, fmt.Errorf("重複のない数字を入力してください")
		}
		seen[num] = true
		guess[i] = num
	}
	return guess, nil
}

// GenerateRandomSecret は重複しない3桁の乱数を生成します
func GenerateRandomSecret() []int {
	digits := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	// math/rand の方は mrand として呼び出す
	mrand.Shuffle(len(digits), func(i, j int) { digits[i], digits[j] = digits[j], digits[i] })
	return digits[:3]
}

// CheckEatBite はEATとBITEを判定します
func CheckEatBite(target, guess []int) (int, int) {
	eat, bite := 0, 0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
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
