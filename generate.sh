#!/bin/bash

set -x -euo pipefail

if [ "x$1" = "x" ]
then
  echo "Please read this script before executing it"
  exit 1
fi

PACKAGE=$1
PREFIX=$2

HOST=${HOST:-apidocs.zerotier.com}

rm -rf ./${PREFIX}
mkdir -p ./${PREFIX}
# github.com/deepmap/oapi-codegen/cmd/oapi-codegen
# note there is an open bug here: https://github.com/deepmap/oapi-codegen/issues/357 related to generation of this code

oapi-codegen -generate types,client -package ${PREFIX} -o ${PREFIX}/gen.go http://${HOST}/${PACKAGE}-v1/api-spec.json
