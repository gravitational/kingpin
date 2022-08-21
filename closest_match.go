package kingpin

import "math"

func GetClosestMatch(arg string, options []string, maxEdits int) string {
	minDistance := math.MaxInt
	closestWord := ""

	for _, word := range options {
		distance := levenshteinDistance(arg, word)
		if distance < minDistance ||
			(distance == minDistance &&
				abs(len(arg)-len(closestWord)) > abs(len(arg)-len(word))) {
			minDistance = distance
			closestWord = word
		}
	}

	if minDistance > maxEdits {
		// If the closes match require more edits than maxEdits returns
		// an empty string.
		return ""
	}

	return closestWord
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func levenshteinDistance(word1, word2 string) int {
	if len(word1) == 0 {
		return len(word2)
	}

	if len(word2) == 0 {
		return len(word1)
	}

	dp := make([][]int, len(word1)+1)
	for i := 0; i < len(dp); i++ {
		dp[i] = make([]int, len(word2)+1)
	}

	for i := 0; i < len(dp); i++ {
		dp[i][0] = i
	}

	for j := 0; j < len(dp[0]); j++ {
		dp[0][j] = j
	}

	for i := 1; i < len(dp); i++ {
		for j := 1; j < len(dp[i]); j++ {
			sameLetter := word1[i-1] == word2[j-1]
			substitutionCost := 0
			if !sameLetter {
				substitutionCost++
			}

			dp[i][j] = min(dp[i][j-1]+1, min(dp[i-1][j]+1, dp[i-1][j-1]+substitutionCost))
		}
	}

	return dp[len(dp)-1][len(dp[0])-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
