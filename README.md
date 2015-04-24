[![wercker status](https://app.wercker.com/status/502464e21d5481a3704b3054ac24c8f7/s "wercker status")](https://app.wercker.com/project/bykey/502464e21d5481a3704b3054ac24c8f7)

# proxy-collector

throw clone request to multiple host and collect responses as json.

## INSTALL

```
$ go get github.com/soh335/proxy-collector/...
```

## USAGE

```
$ proxy-collector -config config.json -host 127.0.0.1 -port 7423
$ curl -s 127.0.0.1:7243/ping | jq . # => (http://localhost:5000/ping, http://localhost:6000/ping)
[
  {
    "target": "http://localhost:5000",
    "body": {
      "pong": "ok"
    },
    "status_code": 200
  },
  {
    "target": "http://localhost:6000",
    "body": {
      "pong": "ok"
    },
    "status_code": 200
  }
]
```

### CONFIG

```json
{
        "target_list": [
                "http://localhost:5000",
                "http://localhost:6000"
        ],
        "body_fallback": 0
}
```

### NO JSON RESPONSE OR INVALID JSON

#### BodyFallbackNone

set empty body

#### BodyFallbackJsonEncode

encode to base64 string and set to body

### LICENSE

MIT
