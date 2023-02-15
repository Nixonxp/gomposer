package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var requiredFiles = []string{
	"go.mod",
	"go.private",
}

type PrivateModule struct {
	url        string
	name       string
	version    string
	mainBranch string
}

type PrivateRepository struct {
	url        string
	mainBranch string
}

func main() {
	checkFilesAndFolders()
	sync()

	fmt.Println("DONE")
}

func sync() {
	var err error
	resultModules := []PrivateModule(nil)
	privateRepos := getPrivateRepos()
	if len(privateRepos) == 0 {
		fmt.Println("Private repositories list is empty")
		return
	}
	modules := getAllExternalModules()

	for _, privateRepoItem := range privateRepos {
		repoName := strings.TrimSuffix(privateRepoItem.url[strings.LastIndex(privateRepoItem.url, "/")+1:], ".git")

		if len(repoName) > 0 {
			if module, ok := getMatchFromArray(repoName, modules); ok {
				version := getModuleVersion(module)
				resultModules = append(resultModules, PrivateModule{
					url:        privateRepoItem.url,
					name:       repoName,
					version:    version,
					mainBranch: privateRepoItem.mainBranch,
				})

			} else {
				panic(fmt.Sprintf("Private repository <%s> does not exist in go.mod \n", repoName))
			}
		} else {
			panic(fmt.Sprintf("Private repository url - <%s>, is invalid \n", privateRepoItem.url))
		}
	}

	if len(resultModules) == 0 {
		panic("Modules to update is empty")
	}

	for _, moduleItem := range resultModules {
		var outbuf, errbuf strings.Builder
		modulePath := fmt.Sprintf("./vendor-private/%s", moduleItem.name)
		// if module is not exist
		if _, err := os.Stat(modulePath); os.IsNotExist(err) {
			cmd := exec.Command("git", "clone", moduleItem.url)
			cmd.Dir = "./vendor-private"
			_, err := cmd.Output()
			if err != nil {
				panic(err.Error())
			}
			fmt.Printf("Module <%s> cloned to <%s> \n", moduleItem.name, modulePath)
		} else {
			// if module is already exist -> update
			// check current version
			currentVersion := checkCurrentModuleVersion(modulePath)
			if currentVersion == moduleItem.version {
				// skip pull
				fmt.Printf("Module <%s> already has been set to version <%s> \n", moduleItem.name, moduleItem.version)
				continue
			}
			// checkout to main branch because pull on tag not worked
			checkoutToBranch(modulePath, moduleItem.name, moduleItem.mainBranch)
			cmd := exec.Command("git", "pull")
			cmd.Stdout = &outbuf
			cmd.Stderr = &errbuf
			cmd.Dir = modulePath
			err := cmd.Run()
			if err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					_ = outbuf.String()
					stderr := errbuf.String()

					fmt.Printf("Module <%s> pull try to <%s> \n", moduleItem.name, modulePath)
					panic(stderr)
				}
				panic(err.Error())
			}

			stdout := outbuf.String()

			fmt.Printf("Module <%s> pulled for update to <%s> \n", moduleItem.name, modulePath)
			fmt.Printf(stdout)
		}

		if err == nil {
			checkoutToVersion(modulePath, moduleItem.name, moduleItem.version)
		}
	}
}

func checkCurrentModuleVersion(modulePath string) string {
	var outbuf, errbuf strings.Builder
	result := ""
	cmd := exec.Command("git", "describe", "--exact-match", "--tags")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	cmd.Dir = modulePath
	err := cmd.Run()

	if err != nil {
		stderr := errbuf.String()
		fmt.Println(stderr)
		fmt.Println("Tag set empty")
		return result
	}

	stdout := outbuf.String()

	return strings.TrimSpace(stdout)
}

func checkoutToVersion(path, name, version string) {
	cmd := exec.Command("git", "checkout", "tags/"+version)
	cmd.Dir = path
	_, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Module <%s> checkout to version <%s> \n", name, version)
}

func checkoutToBranch(path, name, branch string) {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = path
	_, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Module <%s> checkout to branch <%s> for pull updates \n", name, branch)
}

func getModuleVersion(module string) string {
	re := regexp.MustCompile(`\bv.*\b`)
	matches := re.FindStringSubmatch(module)

	if len(matches) <= 0 {
		panic(fmt.Sprintf("Does not parse module version <%s>", module))
	}

	return matches[len(matches)-1]
}

func getMatchFromArray(match string, array []string) (string, bool) {
	for _, item := range array {
		res := strings.Contains(item, match)
		if res == true {
			return item, true
		}
	}

	return "", false
}

func checkFilesAndFolders() {
	for _, fileName := range requiredFiles {
		if _, err := os.Stat("./" + fileName); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				panic(fmt.Sprintf("%s file does not exist", fileName))
			}
		}
	}

	if _, err := os.Stat("./vendor-private"); os.IsNotExist(err) {
		if err = os.Mkdir("./vendor-private", 0755); err != nil {
			panic("vendor-private directory does not create")
		}
	}
}

func getPrivateRepos() []PrivateRepository {
	resultRepos := []PrivateRepository(nil)

	fileScanner, closer := getFileScanner("./go.private")
	defer closer()

	for fileScanner.Scan() {
		repoItemString := strings.TrimSpace(fileScanner.Text())
		repoInfo := strings.Fields(repoItemString)

		if len(repoInfo) < 2 {
			panic(fmt.Sprintf("Invalid private repo record - <%s>, example: <repo url> <main branch>", repoItemString))
		}

		resultRepos = append(resultRepos, PrivateRepository{
			url:        repoInfo[0],
			mainBranch: repoInfo[1],
		})
	}

	return resultRepos
}

func getAllExternalModules() []string {
	fileScanner, closer := getFileScanner("./go.mod")
	defer closer()

	goModulesSection := false

	result := []string(nil)

	for fileScanner.Scan() {
		if fileScanner.Text() == "require (" {
			goModulesSection = true
			continue
		}

		if goModulesSection == true {
			result = append(result, strings.TrimSpace(fileScanner.Text()))
		}

		if fileScanner.Text() == ")" {
			break
		}
	}

	return result
}

func getFileScanner(filename string) (*bufio.Scanner, func()) {
	readFile, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	return fileScanner, func() {
		err := readFile.Close()
		if err != nil {
			panic(fmt.Sprintf("error close %s", filename))
		}
	}
}
