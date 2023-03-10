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
	"sort"
	"strings"
	"time"
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
	SortFilesByModTime            bool
	DataRegExpPattern             string
	ColumnHeaderSpaceSeparated    string
}

func (gc GrepConfig) String() string {
	return fmt.Sprintf("GrepConfig; FilePattern=%q; Recursively=%t; RegExp=%q; ColumnHeader=%q;\n",
		gc.AbsoluteFilePathRegExpPattern, gc.SearchFilesRecursively, gc.DataRegExpPattern,
		gc.ColumnHeaderSpaceSeparated)
}

func getConfig(config *GrepConfig, fpConfig *string) {
	*fpConfig = getUserInput("Setting JSON: ")
	bufConfig, err := os.ReadFile(*fpConfig)
	check(err)
	err = json.Unmarshal(bufConfig, &config)
	check(err)
}

func isFile(dirIn string, files *[]string, fpPattern string) bool {

	// check if file is a folder
	fileInfo, err := os.Stat(dirIn)
	check(err)
	if fileInfo.IsDir() {
		// === Directory ====
		// isFile() returns False
		return false
	} else {
		// === File ===
		// check if file path regular expression matches
		re := regexp.MustCompile(fpPattern)
		if !re.Match([]byte(dirIn)) {
			panic("Pattern Error: fpPattern does not match dirIn!")
		}
		// isFile() returns True (because fpPattern matched)
		return true
	}
}

func getFilesTopOnly(dirIn string, files *[]string, fpPattern string) {
	re := regexp.MustCompile(fpPattern)
	fileInfos, err := ioutil.ReadDir(dirIn)
	check(err)
	for _, f := range fileInfos {
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
	fmt.Printf("%s Start populateDataTable()\n", getCurrentTime())

	// compile the regular expression
	re := regexp.MustCompile(*dataPattern)

	fileCount := len(*files)
	var progress int

	// loop through files
	for i := 0; i < fileCount; i++ {
		// print progress
		progress = (i + 1) * 100 / fileCount
		fmt.Printf("\rProgress %d percent.", progress)

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
	fmt.Printf("\r%s Completed populateDataTable()\n", getCurrentTime())
}

func getCurrentTime() string {
	return time.Now().Format("15:04:05")
}

func getFileModTime(fp string) int64 {
	obj, err := os.Stat(fp)
	check(err)
	return obj.ModTime().Unix()
}

func SortFilesByModTime(files *[]string) {
	sort.Slice(*files, func(i, j int) bool {
		return getFileModTime((*files)[i]) < getFileModTime((*files)[j])
	})
}

func main() {
	// title
	fmt.Println("======= Single Grep ========")

	// get configuration
	var config GrepConfig
	var fpConfig string
	getConfig(&config, &fpConfig)
	// fmt.Printf("Config = %v\n", config)

	// get input folder
	dirIn := getUserInput("Data Folder: ")
	dirOut := getUserInput("Output Folder: ")

	// populate files
	fmt.Printf("%s Populate files\n", getCurrentTime())
	var files []string
	if isFile(dirIn, &files, config.AbsoluteFilePathRegExpPattern) {
		files = append(files, dirIn)
	} else if config.SearchFilesRecursively {
		getFilesRecursively(dirIn, &files, config.AbsoluteFilePathRegExpPattern)
	} else {
		getFilesTopOnly(dirIn, &files, config.AbsoluteFilePathRegExpPattern)
	}
	fmt.Printf("File Count = %d\n", len(files))

	// sort files by mod time
	if config.SortFilesByModTime {
		SortFilesByModTime(&files)
		fmt.Printf("%s Files are sorted by Mod Time.\n", getCurrentTime())
	}

	// get matches
	fmt.Printf("%s Find matches\n", getCurrentTime())
	var matches [][][]uint8
	populateDataTable(&matches, &files, &config.DataRegExpPattern)
	fmt.Printf("%s Match Count = %d\n", getCurrentTime(), len(matches))

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
	fnOut := strings.Split(filepath.Base(fpConfig), ".")[0]
	fpOut := filepath.Join(dirOut, fnOut+".txt")
	fmt.Printf("%s Writing file: %q\n", getCurrentTime(), fpOut)
	os.WriteFile(fpOut, report, 0)
	fmt.Printf("%s Completed\n", getCurrentTime())
}
