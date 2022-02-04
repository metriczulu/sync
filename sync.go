package main

import (
	"fmt"
	"flag"
	"strings"
	_ "regexp"
	"os"
	"bufio"
)

type Config map[string]map[string]string

func ReadConfig(fileLocation string, reverse bool) (Config, error) {
	// function to read a config file and parse it into a Config map
	if fileLocation == "" {
		return map[string]map[string]string {"": map[string]string {"": ""}}, nil
	}
	file, err := os.Open(fileLocation)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currKey := ""
	currLine := ""
	var currLineVals []string
	var firstVal string
	var secondVal string
	configMap := make(map[string]map[string]string)
	for scanner.Scan() {
		currLine = scanner.Text()
		if (len(currLine) > 0) && (currLine[0] == '[') && (currLine[len(currLine)-1] == ']') {
			currKey = strings.TrimSpace(currLine[1:len(currLine)-1])
			configMap[currKey] = make(map[string]string)
		} else if currLineVals = strings.Split(currLine, "="); len(currLineVals) > 1 {
			firstVal = strings.TrimSpace(currLineVals[0])
			secondVal = strings.TrimSpace(currLineVals[1])
			if reverse {
				configMap[currKey][secondVal] = firstVal
			} else {
				configMap[currKey][firstVal] = secondVal
			}
		}
	}
	return configMap, nil
}

func main() {
	configFileLocation := flag.String("config", ".sync", "Sync config file.")
	unsyncFlag := flag.Bool("unsync", true, "Unsync the files.")
	flag.Parse()

	configMap, readErr := ReadConfig(*configFileLocation, true)
	if readErr != nil {
		fmt.Println("Error parsing config file:", readErr)
	}

	fmt.Println(configMap)
	_ = configFileLocation
	_ = unsyncFlag
}