package main

import (
	"fmt"
	// Uncomment this block to pass the first stage!
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {

	// Uncomment this block to pass the first stage!

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	cmd := exec.Command(command, args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	tmpDir := "/tmp/dockerfs"
	err := os.Mkdir(tmpDir, 0744)
	if err != nil {
		//already directory exists
	}
	//fmt.Println("1: ", filepath.Join(tmpDir, filepath.Dir(command)))
	err = exec.Command("mkdir", "-p", filepath.Join(tmpDir, filepath.Dir(command))).Run()
	if err != nil {
		panic("mkdir failed: " + err.Error())
	}
	//fmt.Println("2: ", filepath.Join(tmpDir, command))
	err = exec.Command("cp", command, filepath.Join(tmpDir, command)).Run()
	if err != nil {
		panic("copy failed: " + err.Error())
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: tmpDir,
	}
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Err: %v", err)
		fmt.Println("ProcessState:", cmd.ProcessState)
		os.Exit(cmd.ProcessState.ExitCode())
	}

}
