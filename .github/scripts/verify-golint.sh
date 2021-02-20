#!/bin/bash
golint_output=$(golint $1 2>&1)
golint_exit_code=$?

if [[ "$golint_exit_code" -ne 0 ]]; then
    echo "golint exited with $golint_exit_code"
    echo "${golint_output}"
    exit $golint_exit_code
elif [[ -n "${golint_output}" ]]; then
    echo "golint found errors"
    echo "${golint_output}"
    exit 1
else
    echo "golint didn't find any errors"
fi
