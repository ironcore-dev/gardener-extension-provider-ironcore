#!/usr/bin/env bash

if ! git diff --name-only --exit-code; then
    echo "#####################################################################################################"
    echo "Project is dirty. Make sure you run 'make generate' and 'go mod tidy' before committing your changes."
    echo "#####################################################################################################"
    exit 1
fi
