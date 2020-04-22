#!/bin/sh

# example:
#
#   $ ./curl.sh
#   12345678
#   $

TOKEN="test"
URL_BASE="http://127.0.0.1:8000"

curl -XPOST -H "Authorization: Bearer ${TOKEN}" "${URL_BASE}/new-key-claim"
