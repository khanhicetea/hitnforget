## HIT N FORGET : HTTP LATER SERVER

Send request then forget it, the job queue workers send it later !

## USAGE

Run directly from source

```bash
$ go mod tidy
$ go run main.go
```

Build and run

```bash
$ go mod tidy
$ go build -o hitnforget ./main.go
$ 
```
**Requirements** :

- Local redis server running with localhost:6379 (no password)

## HIT

```bash
curl -X POST localhost:3333/queue \
    -H "X-Hnf-Url: https://mockbin.org/bin/52b0dc2a-a584-45e7-84f2-2ad99741a59a?foo=bar&foo=baz" \
    -H "X-Hnf-Method: POST" \
    -H "X-Men: Wolverine" \
    -d 'data=123&haha=567'
```

With :

- `X-Hnf-Url` : real url you want to hit
- `X-Hnf-Method` : real method you want to hit

It copies all headers (excluded X-Hnf-*), url, query params and body

## NOTICE

It's demo my idea to solve my edge case problem, use as your own risks

## LICENSE

MIT