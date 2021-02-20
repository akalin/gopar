#!/bin/bash
gofmt_output=$(gofmt -l -s $1 2>&1)
gofmt_exit_code=$?

if [[ "$gofmt_exit_code" -ne 0 ]]; then
    echo "gofmt -s exited with $gofmt_exit_code"
    echo "${gofmt_output}"
    exit $gofmt_exit_code
elif [[ -n "${gofmt_output}" ]]; then
    echo "gofmt -s found diffs"
    echo "${gofmt_output}"
    exit 1
else
    echo "gofmt -s didn't find any diffs"
fi
