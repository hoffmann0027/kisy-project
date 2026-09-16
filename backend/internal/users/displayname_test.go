package users

import (
	"errors"
	"testing"
)

func TestNormalizeDisplayName(t *testing.T) {
	ok := map[string]string{
		"Анна Смирнова":        "Анна Смирнова",
		"  Анна   Смирнова  ":  "Анна Смирнова",
		"Anna\tSmith":          "Anna Smith",
		"Jo":                   "Jo",
		"Ёлкин":                "Ёлкин",
		"Łukasz Żółć":          "Łukasz Żółć",
		"Олексій Їжак":         "Олексій Їжак",
		"Ab Cd Ef Gh Ij Kl Mn": "Ab Cd Ef Gh Ij Kl Mn",
		"Аааааааааааааааааааааааааааааааааааааааа": "Аааааааааааааааааааааааааааааааааааааааа", // 40
	}
	for in, want := range ok {
		got, err := NormalizeDisplayName(in)
		if err != nil || got != want {
			t.Errorf("NormalizeDisplayName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	bad := map[string]error{
		"":    ErrDisplayNameLength,
		"   ": ErrDisplayNameLength,
		"Я":   ErrDisplayNameLength,
		"Ааааааааааааааааааааааааааааааааааааааааа": ErrDisplayNameLength, // 41
		"hamza_1":    ErrDisplayNameCharacters,
		"Анна-Мария": ErrDisplayNameCharacters,
		"O'Brien":    ErrDisplayNameCharacters,
		"Иван 2":     ErrDisplayNameCharacters,
		"Anna 😀":     ErrDisplayNameCharacters,
		"Αλέξης":     ErrDisplayNameCharacters, // Greek: outside the alphabet
		"Анна҃":      ErrDisplayNameCharacters, // Cyrillic combining mark
	}
	for in, want := range bad {
		if _, err := NormalizeDisplayName(in); !errors.Is(err, want) {
			t.Errorf("NormalizeDisplayName(%q) error = %v; want %v", in, err, want)
		}
	}
}
