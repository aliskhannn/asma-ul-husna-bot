// Package entities contains domain entities used across the application.
package entities

// Name represents one of the 99 names of Allah from the Asma-ul-Husna.
// It includes the Arabic name, its transliteration, English translation,
// meaning, and audio reference.
type Name struct {
	Number          int    `json:"number"`          // number of the name (from 1 to 99)
	ArabicName      string `json:"name"`            // Arabic name of Allah
	Transliteration string `json:"transliteration"` // transliteration of the Arabic name
	Translation     string `json:"translation"`     // English translation of the name
	Meaning         string `json:"meaning"`         // detailed meaning of the name
	Audio           string `json:"audio"`           // reference to audio file for pronunciation
}
