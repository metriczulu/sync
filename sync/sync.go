package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ConfigMaps map[string]map[string]string // mapping used for normal configs
type ConfigLists map[string][]string
type ReturnVals struct {
	val string
	err error
}

func formatInfo(text string, prefix string) string {
	// looks odd because Go is ac'in' weird around string starts
	text = strings.Replace(text, "\r", "", -1)
	text = prefix + text
	text = strings.Replace(text, "\n", "\n"+prefix, -1)
	return text
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

func overwriteFiles(fileName string, returnVal ReturnVals, writeOutput bool, infoFlag bool, prefix string) {
	if returnVal.err == nil {
		if writeOutput {
			fmt.Println("[info] Updating file:", fileName)
			fmt.Println(returnVal.val)
			file, _ := os.Create(fileName)
			defer file.Close()
			if infoFlag {
				fmt.Println("[info] Writing file:", fileName)
			}
			file.WriteString(returnVal.val)
		} else if infoFlag {
			fmt.Println("[info] Unwritten file change:", fileName)
			fmt.Print("\n" + formatInfo(returnVal.val, prefix) + "\n\n")
		}
	} else {
		fmt.Println("[error] Error processing file", fileName, ":", returnVal.err)
	}
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

func subTokensAndSave(file string, tokens map[string]string, writeOutput bool, infoFlag bool, prefix string, wg sync.WaitGroup) {
	outputText, rawText, outputError := ReadAndSubstituteTokens(file, tokens)
	if outputText != rawText {
		wg.Add(1)
		go func() {
			overwriteFiles(file, ReturnVals{outputText, outputError}, writeOutput, infoFlag, prefix)
			wg.Done()
		}()
	} else {
		if infoFlag {
			fmt.Println("[info] File not modified:", file)
		}
	}
}

func isInList(s string, list []string) bool {
	for _, v := range list {
		if strings.TrimSpace(v) == strings.TrimSpace(s) {
			return true
		}
	}
	return false
}

func IterateFilesAndSubTokens(files []string, tokens map[string]string, acceptedExts []string, ignoredFiles []string, info bool, writeOutput bool, prefix string) {
	// iterate over a list of files and substitute tokens
	var fileExt string
	var wg sync.WaitGroup
	for _, file := range files {
		fileExt = filepath.Ext(file)
		if ((len(acceptedExts) == 0) || isInList(fileExt, acceptedExts) || (fileExt == "")) && (file[0] != '.') && !(isInList(filepath.Base(file), ignoredFiles) || isInList(filepath.Dir(file), ignoredFiles)) {
			if info {
				fmt.Println("[info] Processing file:", file)
			}
			subTokensAndSave(file, tokens, writeOutput, info, prefix, wg)
		}
	}
	wg.Wait()
}

func ReadConfig(fileLocation string, reverse bool) (ConfigMaps, ConfigLists, error) {
	// function to read a config file and parse it into a Config map
	if fileLocation == "" {
		return map[string]map[string]string{"": map[string]string{"": ""}}, map[string][]string{"": []string{}}, nil
	}
	file, err := os.Open(fileLocation)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	listConfigs := []string{"extensions", "ignore", "include"}
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
			currKey = strings.TrimSpace(currLine[1 : len(currLine)-1])
			if !isInList(currKey, listConfigs) {
				configMap[currKey] = make(map[string]string)
			}
		} else if currLineVals = strings.Split(currLine, "="); len(currLineVals) > 1 {
			firstVal = strings.TrimSpace(currLineVals[0])
			secondVal = strings.TrimSpace(currLineVals[1])
			if (isInList(currKey, listConfigs)) || !reverse {
				configMap[currKey][firstVal] = secondVal
			} else {
				configMap[currKey][secondVal] = firstVal
			}
		} else if (len(currLine) > 0) && (len(strings.Split(currLine, "=")) == 1) && (isInList(currKey, listConfigs)) {
			inclusionMap[currKey] = append(inclusionMap[currKey], currLine)
		}
	}
	return configMap, inclusionMap, nil
}

func pprint(s interface{}, prefix string) {
	outSt, err := json.MarshalIndent(s, prefix, "  ")
	if err == nil {
		fmt.Print("\n" + prefix + string(outSt) + "\n\n")
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
	unwriteOutput := flag.Bool("unwrite", false, "Write the output to the files.")
	indentPrefix := flag.String("prefix", "   ", "Indent prefix for verbose display of information.")
	flag.Parse()

	writeOutput := !(*unwriteOutput)
	configMap, inclusionMap, err := ReadConfig(*configFileLocation, *unsyncFlag)
	var filesList []string
	if *infoFlag {
		fmt.Println("[info] Configs:")
		pprint(configMap, *indentPrefix)
		fmt.Println("[info] Config Lists:")
		pprint(inclusionMap, *indentPrefix)
	}
	if err != nil {
		fmt.Println("[error] Error parsing config file:", err)
	}

	// if there are specific files to include listed in the config file, only look at those
	if includes, ok := inclusionMap["include"]; ok {
		filesList = includes
	} else {
		foundFiles, globErr := WalkMatch("./", *filesFlag)
		if globErr != nil {
			fmt.Println("[error] Error globbing files:", globErr)
		}
		filesList = foundFiles
	}

	IterateFilesAndSubTokens(filesList, configMap["tokens"], inclusionMap["extensions"], inclusionMap["ignore"], *infoFlag, writeOutput, *indentPrefix)
}
