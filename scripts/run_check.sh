#!/bin/sh
export GOPROXY=https://goproxy.cn,direct
mkdir -p /tmp/check_app && cd /tmp/check_app
cp /check_skills.go .
go mod init check_app
go get gorm.io/gorm
go get gorm.io/driver/sqlite
go get github.com/lazyautoops/lazy-auto-ops@latest || true
go run check_skills.go
