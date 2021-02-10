#!/bin/bash
gofmt_output=$(gofmt -l -s $1)

if [[ -n "${gofmt_output}" ]]; then
    echo "gofmt -s found diffs"
    echo "${gofmt_output}"
    exit 1
else
    echo "No gofmt -s diffs found"
fi
