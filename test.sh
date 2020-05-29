#!/bin/bash

test_dirs=(
    "internal/core"
)

for dir in "${test_dirs[@]}"; do
    CGO_ENABLED=0 go test -v -count 1 $(pwd)/$dir
done
