#!/bin/sh

set -e

dir=$(pwd)

gen() {
    local package=$1
    
    abigen --bin bin/${package}.bin --abi abi/${package}.abi --pkg=${package} --out=${package}/${package}.go
}

compilegen() {
    local package=$1

    docker run --rm  --user $(id -u) -v "${dir}:/contracts" ethereum/solc:0.8.15-alpine - "/contracts/${package}.sol" -o "/contracts/${package}" --abi --bin --overwrite --optimize
    abigen --bin ${package}/${package}.bin --abi ${package}/${package}.abi --pkg=${package} --out=${package}/${package}.go
}

gen polygonzkevmbridge
compilegen BridgeMessageReceiver