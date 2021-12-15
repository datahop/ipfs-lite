#!/bin/bash

result=${PWD}
GOMODCACHE=$result/mod-cache gomobile bind -v -o datahop.aar -target=android github.com/datahop/ipfs-lite/mobile
