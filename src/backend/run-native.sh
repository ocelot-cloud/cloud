#!/bin/bash

set -e

rm -rf data
go build
PROFILE=NATIVE INITIAL_ADMIN_NAME=admin INITIAL_ADMIN_PASSWORD=password ./backend
