package testutils

import (
	"math/rand"
	"os/exec"
	"strings"
	"testing"
)

func GetSystemLocale(t *testing.T, localeType string) string {
	t.Helper()

	out, err := exec.Command("locale").CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, line := range strings.Fields(string(out)) {
		if strings.Contains(line, localeType) {
			value := strings.Split(line, "=")[1]
			return strings.ReplaceAll(value, "\"", "")
		}
	}

	return ""
}

func GetRandomLocale(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("locale", "-a").CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	// get only UTF-8 locales to match the default encoding value
	var locales []string
	lines := strings.Fields(string(out))
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "utf") {
			locales = append(locales, line)
		}
	}

	return locales[rand.Intn(len(locales))]
}
