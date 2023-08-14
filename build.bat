@echo off
set GOOS=js
set GOARCH=wasm

del assets\wasm\k3d_la_lib.wasm 1>nul 2>&1
mkdir assets\wasm 1>nul 2>&1
go build -o assets\wasm\k3d_la_lib.wasm main.go
pause