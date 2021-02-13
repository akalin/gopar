#!/bin/bash
golint_output=$(golint $1)

if [[ -n "${golint_output}" ]]; then
    echo "${golint_output}"
    exit 1
else
    echo "golint didn't find any errors"
fi
