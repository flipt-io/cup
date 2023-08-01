#!/bin/bash

if ! command -v dagger &> /dev/null
then
  echo "Please install Dagger CLI"
  exit 1
fi

dagger run go run build/main.go "$@"
