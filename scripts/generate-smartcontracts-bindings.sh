#!/bin/bash

# This script generates the smart contracts bindings for the smart contracts
if ! command -v abigen &> /dev/null
then
    echo "abigen binary could not be found"
    exit 1
fi
if ! command -v jq &> /dev/null
then
    echo "jq binary could not be found"
    exit 1
fi


SMC_NAME=$1
OUTPUT_BASE_DIR=etherman/smartcontracts/
GENERATE_FILE_OUTPUT_PATH=${OUTPUT_BASE_DIR}/generated_binding/${SMC_NAME}
GENERATE_FILE_OUTPUT=${GENERATE_FILE_OUTPUT_PATH}/${SMC_NAME}.go
mkdir -p ${GENERATE_FILE_OUTPUT_PATH}
echo "Generating: ${GENERATE_FILE_OUTPUT}"
cat etherman/smartcontracts/json/${SMC_NAME}.json | jq .abi |  abigen --abi - --pkg ${SMC_NAME} --out ${GENERATE_FILE_OUTPUT}