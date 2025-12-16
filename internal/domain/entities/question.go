package entities

type Question struct {
	NameNumber    int
	Type          string // "translation", "transliteration", "meaning", "arabic"
	Question      string
	Options       []string // multiple choice
	CorrectIndex  int
	CorrectAnswer string
}
