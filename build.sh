#!/bin/bash

rm assets/wasm/k3d_smc_lib.wasm
mkdir -p assets/wasm/
GOOS=js GOARCH=wasm go build -o assets\wasm\k3d_smc_lib.wasm main.go