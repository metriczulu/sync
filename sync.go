package main

import (
	"fmt"
	"flag"
	"strings"
	"os"
	"bufio"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

type Config map[string]map[string]string // mapping used for normal configs
type Inclusion map[string][]string
type returnVals struct {
	val string
	err error
}

func printReturn(returned returnVals) {
	if returned.err == nil {
		fmt.Println(returned.val)
	} else {
		fmt.Println(returned.err)
	}
}

func SubstituteTokens(text string, inToken string, outToken string) string {
	// function to substitute tokens in a string
	tokenWraps := []string{" ", "\t", "\n", "\r", "\b", "(", ")", "[", "]", "{", "}"}
	if strings.Contains(text, inToken) {
		// below for-loop implementation is awkward but works
		for _, firstChar := range tokenWraps {
			for _, secondChar := range tokenWraps {
				if (firstChar != "") && (secondChar != "") {
					text = strings.Replace(text, firstChar+inToken+secondChar, firstChar+outToken+secondChar, -1)
				}
			}
		}
		// loop won't catch the case where the text is at the beginning or end of the file
		if inToken == text[:len(inToken)] {
			text = outToken + text[len(inToken):]
		}
		if inToken == text[len(text)-len(inToken):] {
			text = text[:len(text)-len(inToken)] + outToken
		}
		return text
	} else {
		return text
	}
}

func SubstituteTokensIter(tokens map[string]string, text string) string {
	// function to read a file and substitute tokens
	if text == "" {
		return ""
	}
	for token, sub := range tokens {
		text = SubstituteTokens(text, token, sub)
	}
	return text
}

func ReadAndSubstituteTokens(file string, tokens map[string]string) (string, string, error) {
	_, err := os.Stat(file)
	if err == nil {
		file, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println("[error] Error reading file to sync:", err)
		} else {
			return SubstituteTokensIter(tokens, string(file)), string(file), nil
		}
	} else {
		fmt.Println("[error] File not found:", file)
	}
	return "", "", err
}

func isInList(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func IterateFilesAndSubTokens(files []string, tokens map[string]string, acceptedExts []string, ignoredFiles []string, info bool) map[string]returnVals {
	// iterate over a list of files and substitute tokens
	returned := make(map[string]returnVals)
	var fileExt string
	for _, file := range files {
		fileExt = filepath.Ext(file)
		if ((len(acceptedExts) == 0) || isInList(fileExt, acceptedExts)) && (file[0] != '.') && !(isInList(filepath.Base(file), ignoredFiles) || isInList(filepath.Dir(file), ignoredFiles)) {
			if info {
				fmt.Println("[info] Processing file:", file)
			}
			outputText, rawText, outputError := ReadAndSubstituteTokens(file, tokens)
			if outputText != rawText {
				returned[file] = returnVals{outputText, outputError}
			}
		}
	}
	return returned
}

func ReadConfig(fileLocation string, reverse bool) (Config, Inclusion, error) {
	// function to read a config file and parse it into a Config map
	if fileLocation == "" {
		return map[string]map[string]string{"": map[string]string {"": ""}}, map[string][]string{"": []string{}}, nil
	}
	file, err := os.Open(fileLocation)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currKey := ""
	currLine := ""
	var currLineVals []string
	var firstVal string
	var secondVal string
	configMap := make(map[string]map[string]string)
	inclusionMap := make(map[string][]string)
	for scanner.Scan() {
		currLine = scanner.Text()
		if (len(currLine) > 0) && (currLine[0] == '[') && (currLine[len(currLine)-1] == ']') {
			currKey = strings.TrimSpace(currLine[1:len(currLine)-1])
			if !strings.Contains(strings.TrimSpace(currKey), "extensions") {
				configMap[currKey] = make(map[string]string)
			}
		} else if currLineVals = strings.Split(currLine, "="); len(currLineVals) > 1 {
			firstVal = strings.TrimSpace(currLineVals[0])
			secondVal = strings.TrimSpace(currLineVals[1])
			if strings.Contains(strings.TrimSpace(currKey), "settings") || !reverse {
			configMap[currKey][firstVal] = secondVal
			} else {
				configMap[currKey][secondVal] = firstVal
			}
		} else if (len(currLine) > 0) && (len(strings.Split(currLine, "=")) == 1) && (strings.Contains(currKey, "extensions") || (strings.Contains(currKey, "ignore"))) {
			inclusionMap[currKey] = append(inclusionMap[currKey], currLine)
		}
	}
	return configMap, inclusionMap, nil
}

func pprint(s interface{}, prefix string) {
	outSt, err := json.MarshalIndent(s, prefix, "  ")
	if err == nil {
		fmt.Print(prefix)
		fmt.Println(string(outSt))
	}
}

func WalkMatch(root, pattern string) ([]string, error) {
	// shameless stolen from https://stackoverflow.com/questions/55300117/how-do-i-find-all-files-that-have-a-certain-extension-in-go-regardless-of-depth
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func main() {
	configFileLocation := flag.String("config", ".sync", "Sync config file.")
	unsyncFlag := flag.Bool("unsync", false, "Unsync the files.")
	filesFlag := flag.String("file", "**", "File(s) to sync.")
	infoFlag := flag.Bool("verbose", false, "Print info about the sync.")
	flag.Parse()

	configMap, inclusionMap, err := ReadConfig(*configFileLocation, *unsyncFlag)
	if *infoFlag {
		fmt.Println("[info] Configs:")
		pprint(configMap, " ... ")
		fmt.Println("[info] Inclusions:")
		pprint(inclusionMap, " ...")
	}
	if err != nil {
		fmt.Println("[error] Error parsing config file:", err)
	}

	filesList, globErr := WalkMatch("./", *filesFlag)
	if globErr != nil {
		fmt.Println("[error] Error globbing files:", globErr)
	}

	final := IterateFilesAndSubTokens(filesList, configMap["tokens"], inclusionMap["extensions"], inclusionMap["ignore"], *infoFlag)
	for updatedFile, returnVal := range final {
		// fmt.Println("****\nUpdated file:", updatedFile)
		// printReturn(returnVal)
		if returnVal.err == nil {
			if *infoFlag {
				fmt.Println("[info] Writing file:", updatedFile)
			}
			file, err := os.Create(updatedFile)
			if err != nil {
				return
			}
			defer file.Close()
			file.WriteString(returnVal.val)
		} else {
			fmt.Println("[error] Error processing file:", updatedFile, ":", returnVal.err)
		}
	}
	_ = configFileLocation
	_ = unsyncFlag
}