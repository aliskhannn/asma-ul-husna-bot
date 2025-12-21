package entities

// Question represents a quiz question for the 99 Names of Allah.
// It includes the name number, question type, the question text, answer options,
// the correct answer index, and the correct answer text.
type Question struct {
	NameNumber    int      // number of the associated name (from 1 to 99)
	Type          string   // type of question: "translation", "transliteration", "meaning", or "arabic"
	Question      string   // the text of the question
	Options       []string // multiple choice answer options
	CorrectIndex  int      // index of the correct answer in the Options slice
	CorrectAnswer string   // the correct answer text
}
