package main

import (
	"bufio"
	. "fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func helmInit(context string) {
	Printf("Helm Init...")

	checkKubectlExists()
	checkHelmExists()
	getFileFromURL("templates/tiller-rbac.yaml","https://k8s-cloud-templates.s3.amazonaws.com/tiller-rbac.yaml")

	cmd := exec.Command("kubectl", "config", "use-context", context)
	failOnError(cmd)

	cmd = exec.Command("kubectl", "apply", "-f", "templates/tiller-rbac.yaml")
	failOnError(cmd)

	cmd = exec.Command("helm", "init", "--service-account", "tiller", "--kube-context", context)
	failOnError(cmd)
	Printf("Helm init completed successfully..")
}

func helmRepoAdd(repo helmRepository) {
	Printf("Adding %s helm repo...\n", string(repo.name))
	if repo.username == "" {
		cmd := exec.Command("helm", "repo", "add", repo.name, repo.url)
		failOnError(cmd)

	} else {
		cmd := exec.Command("helm", "repo", "add", repo.name, repo.url, "--username", repo.username, "--password", repo.password)
		failOnError(cmd)
	}

}

func helmRepoUpdate() {
	Printf("Helm Repo Update...")
	cmd := exec.Command("helm", "repo", "update")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	Printf("output:\n%s", string(out))
}

func helmInstallRelease(release helmRelease, context string) {
	Printf("Installing %s\n", release.name)
	if release.namespace == "" {
		release.namespace = "default"
	}

	cmd := exec.Command(
		"helm", "install",
		"--name", release.name,
		"--namespace", release.namespace,
		"--version", release.version,
		"--kube-context", context,
		release.chart,
	)
	failOnError(cmd)
}

func helmUpgradeRelease(release helmRelease, context string) {
	Printf("Installing %s\n", release.name)
	if release.namespace == "" {
		release.namespace = "default"
	}

	if fileExists(release.valuesfile) {
		cmd := exec.Command(
			"helm", "upgrade",
			release.name,
			"--namespace", release.namespace,
			"--version", release.version,
			"--kube-context", context,
			"--values", release.valuesfile,
			"--install", release.chart,
		)
		failOnError(cmd)

	} else if release.valuesfile == "" {
		cmd := exec.Command(
			"helm", "upgrade",
			release.name,
			"--namespace", release.namespace,
			"--version", release.version,
			"--kube-context", context,
			"--install", release.chart,
		)
		failOnError(cmd)

	} else {
		Printf("Error: Values file %s does not exist\n", release.valuesfile)
		os.Exit(1)
	}

	//if release.valuesfile == "" {
	//	cmd := exec.Command(
	//		"helm", "upgrade",
	//		release.name,
	//		"--namespace", release.namespace,
	//		"--version", release.version,
	//		"--kube-context", context,
	//		"--install", release.chart,
	//	)
	//	failOnError(cmd)
	//
	//}
}

func failOnError(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		Printf(scanner.Text())
		os.Exit(1)
	}
}

func helmAddRepositories(config HelmConfig) {
	for i := 0; i < len(config.Repositories); i++ {
		repository := helmRepository{
			name:     config.Repositories[i].Name,
			url:      config.Repositories[i].Url,
			username: config.Repositories[i].Username,
			password: config.Repositories[i].Password,
		}
		helmRepoAdd(repository)
	}
	helmRepoUpdate()
}


func helmInstallReleases(config HelmConfig, context string) {
	for i := 0; i < len(config.Releases); i++ {

		release := helmRelease{
			name:       config.Releases[i].Name,
			namespace:  config.Releases[i].Namespace,
			chart:      config.Releases[i].Chart,
			version:    config.Releases[i].Version,
			valuesfile: config.Releases[i].ValuesFile,
		}
		//releaseStatus := checkIfReleaseExists(release.name)
		//if releaseStatus == 0 {
		//	Printf("Release %s Alreasy Exists, Upgrading...\n", release.name)
		//	helmUpgradeRelease(release, context)
		//} else {
		//	Printf("Release %s doesn't Exists...\n", release.name)
		//	helmInstallRelease(release, context)
		//}
		helmUpgradeRelease(release, context)
	}
}

func checkIfReleaseExists(name string) int {
	Printf("Checking if release:%s already exists\n", name)
	cmd := exec.Command("helm", "history", name)
	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start: %v", err)
		return -1
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				log.Printf("Exit Status: %d", status.ExitStatus())
			}
		} else {
			log.Fatalf("cmd.Wait: %v", err)
		}
	}

	//if err := cmd.Run(); err != nil {
	//	if _, ok := err.(*exec.ExitError); ok {
	//		return -1
	//	}
	//}
	return 0
}
