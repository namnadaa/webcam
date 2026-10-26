// Package i18n translates user-facing text. English is the source language:
// texts are written in English in the code and looked up in the translation of the selected language.
package i18n

import (
	"fmt"
	"strings"
)

// Lang is a language of the program interface.
type Lang int

const (
	English Lang = iota
	Russian
)

var current = English

// translations maps English texts to their translations, per language.
var translations = map[Lang]map[string]string{
	Russian: ru,
}

// Set selects the interface language.
func Set(l Lang) {
	current = l
}

// Current returns the selected interface language.
func Current() Lang {
	return current
}

// T returns the text in the selected language; untranslated texts stay in English.
func T(s string) string {
	if t, ok := translations[current][s]; ok {
		return t
	}
	return s
}

// Tf formats the translated format string.
func Tf(format string, args ...any) string {
	return fmt.Sprintf(T(format), args...)
}

// Ask asks for the interface language; English is chosen on Enter or any other input.
// It is the first question of the program, so it is shown in both languages.
func Ask() {
	fmt.Println("Language / Язык:")
	fmt.Println("[1] English (default)")
	fmt.Println("[2] Русский")
	fmt.Print("Select / Выберите [1-2, Enter = English]: ")

	var input string
	fmt.Scanln(&input)

	if strings.TrimSpace(input) == "2" {
		Set(Russian)
		return
	}
	Set(English)
}
