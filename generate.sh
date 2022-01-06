#!/bin/bash

set -euo pipefail

if [ $# != 2 ]
then
  echo "Please read this script before executing it"
  exit 1
fi

PACKAGE=$1
SPEC=$2

HOST=${HOST:-docs.zerotier.com}

rm -rf ./${PACKAGE}
mkdir -p ./${PACKAGE}
# github.com/deepmap/oapi-codegen/cmd/oapi-codegen
# note there is an open bug here: https://github.com/deepmap/oapi-codegen/issues/357 related to generation of this code

oapi-codegen -generate types,client -package ${PACKAGE} -o ${PACKAGE}/gen.go https://${HOST}/openapi/${SPEC}v1.json
