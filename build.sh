#!/bin/bash
GOOS=windows GOARCH=amd64 gogio -buildmode=exe -icon=appicon.png -arch=amd64 -target=windows -o uncors.exe .