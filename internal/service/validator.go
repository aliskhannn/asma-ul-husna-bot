package service

import (
	"strings"
)

// AnswerValidator validates user answers with fuzzy matching support.
type AnswerValidator struct {
	threshold float64 // Similarity threshold (0.0 - 1.0)
}

// NewAnswerValidator creates a new AnswerValidator.
func NewAnswerValidator() *AnswerValidator {
	return &AnswerValidator{
		threshold: 0.8, // 80% similarity required
	}
}

// Validate checks if the user's answer matches the correct answer.
func (v *AnswerValidator) Validate(userAnswer, correctAnswer string) bool {
	// Normalize both strings
	user := v.normalize(userAnswer)
	correct := v.normalize(correctAnswer)

	// Exact match
	if user == correct {
		return true
	}

	// Fuzzy match using Levenshtein distance
	similarity := v.similarity(user, correct)
	return similarity >= v.threshold
}

// normalize normalizes a string for comparison.
func (v *AnswerValidator) normalize(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Trim spaces
	s = strings.TrimSpace(s)

	// Normalize Arabic text
	s = normalizeArabic(s)

	// Remove extra whitespace
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// similarity calculates the similarity between two strings using Levenshtein distance.
func (v *AnswerValidator) similarity(s1, s2 string) float64 {
	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// normalizeArabic normalizes Arabic text by removing diacritics and normalizing characters.
func normalizeArabic(s string) string {
	// Remove Arabic diacritics (harakat)
	s = strings.Map(func(r rune) rune {
		// Arabic diacritics range: U+064B to U+065F
		if r >= 0x064B && r <= 0x065F {
			return -1 // Remove diacritic
		}
		// Tatweel (kashida): U+0640
		if r == 0x0640 {
			return -1
		}
		return r
	}, s)

	// Normalize common Arabic character variations
	replacements := map[rune]rune{
		'أ': 'ا', // Alef with hamza above
		'إ': 'ا', // Alef with hamza below
		'آ': 'ا', // Alef with madda
		'ة': 'ه', // Teh marbuta to heh
		'ى': 'ي', // Alef maksura to yeh
	}

	s = strings.Map(func(r rune) rune {
		if normalized, ok := replacements[r]; ok {
			return normalized
		}
		return r
	}, s)

	return s
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)

	// Create a 2D array for dynamic programming
	rows := len(r1) + 1
	cols := len(r2) + 1

	// Use two rows instead of full matrix for space optimization
	prev := make([]int, cols)
	curr := make([]int, cols)

	// Initialize first row
	for j := 0; j < cols; j++ {
		prev[j] = j
	}

	// Calculate distances
	for i := 1; i < rows; i++ {
		curr[0] = i

		for j := 1; j < cols; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			curr[j] = min(
				curr[j-1]+1,    // Insertion
				prev[j]+1,      // Deletion
				prev[j-1]+cost, // Substitution
			)
		}

		// Swap rows
		prev, curr = curr, prev
	}

	return prev[cols-1]
}
