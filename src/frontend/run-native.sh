#!/bin/bash

VITE_APP_PROFILE="NATIVE" npm run serve
echo -ne "\x1b[?25h" > /dev/null
echo -ne "\x1b[0m"