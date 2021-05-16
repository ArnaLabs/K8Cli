package models

import (
	"fmt"
	"log"
	"os/exec"
)

type helm_release struct {
	name        string
	version     string
	chart_name  string
	values_file string
}

type helm_repository struct {
	name     string
	url      string
	username string
	password string
}

func helm_add_repo(repo helm_repository) {
	// var cmd
	if repo.username == "" {
		cmd := exec.Command("helm", "repo", "add", repo.name, repo.url)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}
		fmt.Printf("output:\n%s", string(out))
	} else {
		cmd := exec.Command("helm", "repo", "add", repo.name, repo.url, "--username", repo.username, "--password", repo.password)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}
		fmt.Printf("output:\n%s", string(out))
	}

}
