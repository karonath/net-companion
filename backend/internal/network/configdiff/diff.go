// Package configdiff compare deux configurations (running vs startup) via SSH.
package configdiff

import "strings"

// DiffLine est une ligne du diff : Op ∈ "same" | "add" | "del".
type DiffLine struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// Diff compare oldText et newText ligne à ligne (algorithme LCS).
func Diff(oldText, newText string) []DiffLine {
	a := splitLines(oldText)
	b := splitLines(newText)
	m, n := len(a), len(b)

	// dp[i][j] = longueur de la LCS de a[:i] et b[:j].
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var rev []DiffLine
	i, j := m, n
	for i > 0 && j > 0 {
		switch {
		case a[i-1] == b[j-1]:
			rev = append(rev, DiffLine{Op: "same", Text: a[i-1]})
			i--
			j--
		case dp[i][j-1] >= dp[i-1][j]:
			// À égalité de LCS, on privilégie l'ajout pour que, sur une ligne
			// modifiée, la suppression apparaisse avant l'ajout (del puis add).
			rev = append(rev, DiffLine{Op: "add", Text: b[j-1]})
			j--
		default:
			rev = append(rev, DiffLine{Op: "del", Text: a[i-1]})
			i--
		}
	}
	for i > 0 {
		rev = append(rev, DiffLine{Op: "del", Text: a[i-1]})
		i--
	}
	for j > 0 {
		rev = append(rev, DiffLine{Op: "add", Text: b[j-1]})
		j--
	}

	// rev est en ordre inverse : on retourne.
	out := make([]DiffLine, len(rev))
	for k := range rev {
		out[k] = rev[len(rev)-1-k]
	}
	return out
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
