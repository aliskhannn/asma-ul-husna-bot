package entities

// Name represents one of the 99 names of Allah from the Asma-ul-Husna.
// It includes the Arabic name, its transliteration, English translation,
// meaning, and audio reference.
type Name struct {
	Number          int    `json:"number"` // number of the name (from 1 to 99)
	ArabicName      string `json:"name"`
	Transliteration string `json:"transliteration"`
	Translation     string `json:"translation"`
	Meaning         string `json:"meaning"`
	Audio           string `json:"audio"`
}
