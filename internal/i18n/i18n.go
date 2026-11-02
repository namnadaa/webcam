// Package i18n translates user-facing text. English is the source language:
// texts are written in English in the code and looked up in the translation of the selected language.
package i18n

import (
	"fmt"
	"io"
	"os"
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

// Ask asks for the interface language until the answer is valid; Enter selects English.
// It is the first question of the program, so it is shown in both languages.
func Ask() {
	fmt.Println("Language / Язык:")
	fmt.Println("[1] English (default)")
	fmt.Println("[2] Русский")

	for {
		fmt.Print("Select / Выберите [1-2, Enter = English]: ")

		input, ok := readLine(os.Stdin)
		if !ok { // input closed, nothing more to ask
			Set(English)
			return
		}

		if lang, valid := parseLang(input); valid {
			Set(lang)
			return
		}
		fmt.Println("Enter 1, 2 or press Enter / Введите 1, 2 или нажмите Enter")
	}
}

// parseLang returns the language for an answer to Ask; an empty answer means English.
func parseLang(input string) (Lang, bool) {
	switch strings.TrimSpace(input) {
	case "", "1":
		return English, true
	case "2":
		return Russian, true
	}
	return English, false
}

// readLine reads one line byte by byte, so nothing after it is consumed
// and the following fmt.Scan calls see the rest of the input.
// Returns false at the end of the input when nothing was read.
func readLine(r io.Reader) (string, bool) {
	var line []byte
	b := make([]byte, 1)

	for {
		n, err := r.Read(b)
		if n == 1 {
			if b[0] == '\n' {
				return string(line), true
			}
			line = append(line, b[0])
		}
		if err != nil {
			return string(line), len(line) > 0
		}
	}
}
