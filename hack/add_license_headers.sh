#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

echo "> Adding Apache License header to all go files where it is not present"

YEAR=$1
if [[ -z "$1" ]]; then
  cat << EOF
Unspecified 'YEAR' argument.
Usage: add_licence_headers.sh <YEAR>
EOF
  exit 1
fi

temp_file=$(mktemp)
trap "rm -f $temp_file" EXIT
sed "s/{YEAR}/${YEAR}/g" hack/license_boilerplate.txt > $temp_file

# Uses the tool https://github.com/google/addlicense
addlicense \
  -f $temp_file \
  -ignore ".idea/**" \
  -ignore ".vscode/**" \
  -ignore "**/*.md" \
  -ignore "**/*.yaml" \
  -ignore "**/Dockerfile" \
  .
