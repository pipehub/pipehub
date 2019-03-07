# HTTPWay
An programmable proxy server.  

Please, don't use it in production *yet*! It's no where near stable and changing too much. But feel free to test and give suggestions.

## Why?
Software development is getting harder and harder, on a typical medium/large solution there are lot's of servers at the request path: API Gateways, Load balancers, Cache servers, Proxies, and Firewalls, just to name a few. These generate latency and require much more engineering/monitoring to do it right.

The core idea of this project is to do more with less. This project is a programmable proxy, so users can extend and customize the software as needed. Features found in other servers will and can be added with Go packages instead of actual servers.

## How?
The code is extended with a thing called `handler`. It's a plain old Go code that is injected at compile time at the application. Being a Go project gives a much higher flexibility at the handler because it can really be anything.

Bellow a configuration sample:
```hcl
server {
  http {
    port = 80
  }
}

host {
  endpoint = "google"
  origin   = "https://www.google.com"
  handler  = "base.Default"
}

handler {
  path    = "github.com/httpway/handler"
  version = "v0.5.0"
  alias   = "base"
}
```

The handler points to the place where the Go code is, it should be a `go gettable` project. A handler is a generic processor that can be used on multiple hosts. A host track the endpoint the proxy gonna listen, where the origin is, and which handler gonna be used to process the requests. Don't forget to add this line at your `/etc/hosts`: `127.0.0.1 google` if you use this config snipped to test.

A real example of a handler can be found [here](https://github.com/httpway/handler).

## How to run it?
First, create a config file:
```bash
cp cmd/httpway/httpway/httpway.sample.hcl cmd/httpway/httpway/httpway.hcl
# edit cmd/httpway/httpway/httpway.hcl
```

Generate the binary:
```bash
make generate
```

Execute it:
```
./cmd/httpway/httpway start -c ./cmd/httpway/httpway.hcl
```