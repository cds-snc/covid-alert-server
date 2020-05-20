# Examples

Receiving a new key is a simple HTTP call to the server with an Authentication header. This should be straightforward to implement as part of any public health worker system. Please note to keep the authentication token secret and avoid exposing generated keys to people who have not have had COVID positive test results

```sh
TOKEN="test"
URL_BASE="http://127.0.0.1:8000"

curl -XPOST -H "Authorization: Bearer ${TOKEN}" "${URL_BASE}/new-key-claim"
```
