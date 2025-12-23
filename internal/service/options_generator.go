package service

import (
	"math/rand"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

// OptionGenerator generates multiple choice options for quiz questions.
type OptionGenerator struct {
	allNames []*entities.Name
}

// NewOptionGenerator creates a new option generator.
func NewOptionGenerator(allNames []*entities.Name) *OptionGenerator {
	return &OptionGenerator{
		allNames: allNames,
	}
}

// GenerateOptions creates 4 multiple choice options including the correct answer.
// Returns: options slice and the index of the correct answer (0-3).
func (g *OptionGenerator) GenerateOptions(
	correctName *entities.Name,
	questionType entities.QuestionType,
) ([]string, int) {
	options := make([]string, 4)

	// Get the correct answer based on question type
	var correctAnswer string
	switch questionType {
	case entities.QuestionTypeTranslation:
		correctAnswer = correctName.ArabicName
	case entities.QuestionTypeTransliteration:
		correctAnswer = correctName.Translation
	case entities.QuestionTypeMeaning:
		correctAnswer = correctName.Transliteration
	case entities.QuestionTypeArabic:
		correctAnswer = correctName.Translation
	default:
		correctAnswer = correctName.Translation
	}

	// Generate 3 wrong options
	wrongOptions := g.generateWrongOptions(correctName, questionType, 3)

	// Randomly place the correct answer
	correctIndex := rand.Intn(4)

	// Fill options array
	wrongIdx := 0
	for i := 0; i < 4; i++ {
		if i == correctIndex {
			options[i] = correctAnswer
		} else {
			options[i] = wrongOptions[wrongIdx]
			wrongIdx++
		}
	}

	return options, correctIndex
}

// generateWrongOptions creates wrong answer choices that are different from the correct one.
func (g *OptionGenerator) generateWrongOptions(correctName *entities.Name, questionType entities.QuestionType, count int) []string {
	wrongOptions := make([]string, 0, count)
	usedNumbers := map[int]bool{correctName.Number: true}

	// Create a pool of candidates
	candidates := make([]*entities.Name, 0, len(g.allNames))
	for _, name := range g.allNames {
		if name.Number != correctName.Number {
			candidates = append(candidates, name)
		}
	}

	// Shuffle candidates
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Pick wrong options
	for _, candidate := range candidates {
		if len(wrongOptions) >= count {
			break
		}

		if usedNumbers[candidate.Number] {
			continue
		}

		var optionText string
		switch questionType {
		case entities.QuestionTypeTranslation:
			optionText = candidate.ArabicName
		case entities.QuestionTypeTransliteration:
			optionText = candidate.Translation
		case entities.QuestionTypeMeaning:
			optionText = candidate.Transliteration
		case entities.QuestionTypeArabic:
			optionText = candidate.Translation
		default:
			optionText = candidate.Translation
		}

		// Avoid duplicates
		isDuplicate := false
		for _, existing := range wrongOptions {
			if existing == optionText {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			wrongOptions = append(wrongOptions, optionText)
			usedNumbers[candidate.Number] = true
		}
	}

	// If we couldn't find enough unique options, add generic ones
	for len(wrongOptions) < count {
		wrongOptions = append(wrongOptions, "Вариант "+string(rune('A'+len(wrongOptions))))
	}

	return wrongOptions
}
