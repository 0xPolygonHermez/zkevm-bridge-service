#!/bin/bash

# This script generates the smart contracts bindings for the smart contracts
if ! command -v abigen &> /dev/null
then
    echo "abigen binary could not be found"
    exit 1
fi

OUTPUT_BASE_DIR=../etherman/smartcontracts/

set -e

gen() {
    local package=$1

    mkdir -p ${OUTPUT_BASE_DIR}/${package}

    abigen --bin ${OUTPUT_BASE_DIR}/bin/${package}.bin --abi ${OUTPUT_BASE_DIR}/abi/${package}.abi --pkg=${package} --out=${OUTPUT_BASE_DIR}/${package}/${package}.go
}

gen_from_json(){
    local package=$1
     mkdir -p ${OUTPUT_BASE_DIR}/${package}
     abigen --combined-json ${OUTPUT_BASE_DIR}/json/${package}.json --pkg=${package} --out=${OUTPUT_BASE_DIR}/${package}/${package}.go
}

#gen_from_json pingreceiver
gen claimcompressor