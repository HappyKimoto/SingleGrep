package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getUserInput(promptMessage string) string {
	// get user input
	fmt.Print(promptMessage)
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')
	check(err)
	// remove CR, LF, and double quotes
	userInput = strings.ReplaceAll(userInput, "\r\n", "")
	userInput = strings.ReplaceAll(userInput, "\"", "")
	// return clean user input
	return userInput
}

type GrepConfig struct {
	AbsoluteFilePathRegExpPattern string
	SearchFilesRecursively        bool
	DataRegExpPattern             string
	ColumnHeaderSpaceSeparated    string
	OutputFileName                string
}

// {
//     "AbsoluteFilePathRegExpPattern": ".*txt",
//     "SearchFilesRecursively": true,
//     "DataRegExpPattern": "(..../../.. ..:..:..) ([a-zA-Z]+)",
//     "ColumnHeaderSpaceSeparated": "DateTime Event",
//     "OutputFileName": "DateTimeEvent.txt"
// }

func (gc GrepConfig) String() string {
	return fmt.Sprintf("GrepConfig; FilePattern=%q; Recursively=%t; RegExp=%q; ColumnHeader=%q; OutFile=%q\n",
		gc.AbsoluteFilePathRegExpPattern, gc.SearchFilesRecursively, gc.DataRegExpPattern,
		gc.ColumnHeaderSpaceSeparated, gc.OutputFileName)
}

func getConfig(config *GrepConfig) {
	fpConfig := getUserInput("Setting JSON: ")
	bufConfig, err := os.ReadFile(fpConfig)
	check(err)
	err = json.Unmarshal(bufConfig, &config)
	check(err)
}

func getFilesTopOnly(dirIn string, files *[]string, fpPattern string) {
	re := regexp.MustCompile(fpPattern)
	fileInfos, err := ioutil.ReadDir(dirIn)
	check(err)
	for n, f := range fileInfos {
		fmt.Printf("n=%d\n", n)
		if !f.IsDir() {
			path := filepath.Join(dirIn, f.Name())
			if re.MatchString(path) {
				*files = append(*files, path)
			}
		}
	}
}

func getFilesRecursively(dirIn string, files *[]string, fpPattern string) {
	re := regexp.MustCompile(fpPattern)
	err := filepath.Walk(dirIn,
		func(path string, info os.FileInfo, err error) error {
			// error handling
			if err != nil {
				return err
			}
			// if file is not a directory and file pattern matches, populate the file list
			if !info.IsDir() {
				if re.MatchString(path) {
					*files = append(*files, path)
				}
			}
			// return okay
			return nil
		})
	check(err)
}

func populateDataTable(matches *[][][]uint8, files *[]string, dataPattern *string) {
	// compile the regular expression
	re := regexp.MustCompile(*dataPattern)

	// loop through files
	for i := 0; i < len(*files); i++ {
		// read file
		contents, err := os.ReadFile((*files)[i])
		check(err)
		// get matches
		matchesCurrent := re.FindAllSubmatch(contents, -1)
		// append if found any
		if len(matchesCurrent) > 0 {
			*matches = append(*matches, matchesCurrent...)
		}
	}
}

func main() {
	// title
	fmt.Println("======= Single Grep ========")

	// get configuration
	var config GrepConfig
	getConfig(&config)
	fmt.Printf("Config = %v\n", config)

	// get input folder
	dirIn := getUserInput("Data Folder: ")
	dirOut := getUserInput("Output Folder: ")

	// populate file list
	var files []string

	// populate files recursively or top only
	if config.SearchFilesRecursively {
		getFilesRecursively(dirIn, &files, config.AbsoluteFilePathRegExpPattern)
	} else {
		getFilesTopOnly(dirIn, &files, config.AbsoluteFilePathRegExpPattern)
	}

	fmt.Printf("File Count = %d\n", len(files))

	// get matches
	var matches [][][]uint8
	populateDataTable(&matches, &files, &config.DataRegExpPattern)
	fmt.Printf("Match Count = %d\n", len(matches))

	const firstColumnIndex = 1
	var sepCol = []byte("\t")  // 0x09 = Tab
	var sepLine = []byte("\n") // 0x0A = Line feed

	// get header from space separated to tab separated with ending line feed in bytes
	report := []byte(strings.ReplaceAll(config.ColumnHeaderSpaceSeparated, " ", "\t") + "\n")

	// create otuput body
	for i := 0; i < len(matches); i++ {
		// append line
		line := bytes.Join(matches[i][firstColumnIndex:], sepCol)
		report = append(report, line...)
		// append line separator
		report = append(report, sepLine...)
	}

	fpOut := filepath.Join(dirOut, config.OutputFileName)
	os.WriteFile(fpOut, report, 0)
}
