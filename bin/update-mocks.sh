#!/bin/bash

docker run -v "$PWD":/src -w /src vektra/mockery --all --keeptree