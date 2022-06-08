package main

import (
	"fmt"
	"os"
	"testing"
)

func TestSubstituteTokens(t *testing.T) {
	expectedText := "SUB of the SUB Test emergency. shane."
	tokenMap := map[string]string{"test": "SUB"}
	outputText := SubstituteTokensIter(tokenMap, "test of the test Test emergency. shane.")
	if expectedText != outputText {
		t.Errorf("Expected: %s but got %s", outputText, expectedText)
	}
}

func TestReadAndSub(t *testing.T) {
	file, err := os.CreateTemp("", "temp_test.*.py")
	fmt.Println("Created temp file: ", file.Name())
	if err != nil {
		fmt.Println(err)
	}
	defer os.Remove(file.Name())
	initialText := `from shane import dog\n\n` +
		`print(dog.speak())`
	expectedSubbedText := `from pet import dog\n\n` +
		`print(dog.speak())`
	file.WriteString(initialText)
	file.Seek(0, 0)
	subbedText, originalText, _ := ReadAndSubstituteTokens(file.Name(), map[string]string{"shane": "pet"})
	if err != nil {
		fmt.Println(err)
	}
	if originalText != initialText {
		fmt.Println("Initial Text: ")
		t.Errorf("Expected: %s but got %s", initialText, originalText)
	}
	if subbedText != expectedSubbedText {
		fmt.Println("Generated Text: ")
		t.Errorf("Expected: %s but got %s", expectedSubbedText, subbedText)
	}
}
