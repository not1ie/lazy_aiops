#!/bin/sh
export GOPROXY=https://goproxy.cn,direct
mkdir -p /tmp/test_api_2 && cd /tmp/test_api_2
cat << 'INNER_EOF' > main.go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/api/v1/ai/skills-public", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Response status:", resp.Status)
	fmt.Println("Response body:", string(body))
}
INNER_EOF
go mod init test_api_2
go run main.go
