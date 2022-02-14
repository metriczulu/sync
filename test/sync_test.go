package main

include (
	"testing"
	_ "strings"
)

func TestSubstituteTokens(t *testing.T) {
	expectedText := "SUB of the SUB Test emergency. SUB."
	tokenMap := map[string]string{"test": "SUB"}
	outputText := SubstituteTokensIter(tokenMap, "test of the test Test emergency. test.")
	if inputTex != outputText {
		t.Errorf("Expected: %s but got %s", expectedText, outputText)
	}
}