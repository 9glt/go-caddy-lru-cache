# go-caddy-lru-cache

```
go get -u github.com/caddyserver/xcaddy/cmd/xcaddy
xcaddy build  --with github.com/9glt/go-caddy-lru-cache
```


Caddyfile
```
http://*:1200 {

	route {
		tscache .ts
		reverse_proxy http://127.0.0.1
	}
}
```

```
./caddy run
```
