package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type TokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	Expires     int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type Manifest struct {
	Name     string     `json:"name"`
	Tag      string     `json:"tag"`
	FSLayers []fsLayers `json:"fsLayers"`
}

type fsLayers struct {
	BlobSum string `json:"blobSum"`
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {

	img := os.Args[2]
	split := strings.Split(img, ":")

	repo := "library"
	image := split[0]

	tag := "latest"
	if len(split) == 2 {
		tag = split[1]
	}

	request, err := http.NewRequest("GET", fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s/%s:pull", repo, image), nil)
	if err != nil {
		fmt.Printf("Err: %v", err)
		os.Exit(1)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)

	var result TokenResponse
	json.NewDecoder(resp.Body).Decode(&result)

	manifestReq, err := http.NewRequest("GET", fmt.Sprintf("https://registry.hub.docker.com/v2/%s/%s/manifests/%s", repo, image, tag), nil)
	if err != nil {
		fmt.Printf("Err: %v", err)
		os.Exit(1)
	}
	manifestReq.Header.Add("Authorization", "Bearer "+strings.TrimSpace(result.Token))
	manifestReq.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v1+json")

	mani, err := http.DefaultClient.Do(manifestReq)
	if err != nil {
		fmt.Printf("Err: %v", err)
		os.Exit(1)
	}
	var manifest Manifest
	json.NewDecoder(mani.Body).Decode(&manifest)

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	tmpDir := "/tmp/dockerfs"

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     tmpDir,
		Cloneflags: syscall.CLONE_NEWPID,
	}

	err = os.Mkdir(tmpDir, 0744)
	if err != nil {
		//already directory exists
	}

	for _, value := range manifest.FSLayers {
		req, err := http.NewRequest("GET", "https://registry-1.docker.io/v2/library/"+image+"/blobs/"+value.BlobSum, nil)
		if err != nil {
			fmt.Println("er1")
		}
		req.Header.Add("Authorization", "Bearer "+strings.TrimSpace(result.Token))
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("er2")
		}
		defer resp.Body.Close()
		f, e := os.Create(tmpDir + "/output")
		if e != nil {
			panic(e)
		}
		defer f.Close()
		f.ReadFrom(resp.Body)

		_, err = exec.Command("tar", "xf", tmpDir+"/output", "-C", tmpDir).Output()
		if err != nil {
			fmt.Printf("OUT ERR untar => %+v", err)
		}
		os.RemoveAll(tmpDir + "/output")
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

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Err: %v", err)
		fmt.Println("ProcessState:", cmd.ProcessState)
		os.Exit(cmd.ProcessState.ExitCode())
	}

}
