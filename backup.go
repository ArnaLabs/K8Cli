package main

import (
	. "fmt"
	"os/exec"
)

func takeBackup(name string, context string) {
	Printf("Creating backup %s on %s cluster...\n", name, context)

	//velero backup create backup4 --wait --kubecontext 'test-eks5'
	cmd := exec.Command("velero", "backup", "create", name, "--wait", "--kubecontext", context)
	failOnError(cmd)

}
