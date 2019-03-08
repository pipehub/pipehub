# Pipehub
A programmable proxy server.  
Please, don't use it in production **yet**! It's nowhere near stable and changing too much.

## Why?
Software development is getting harder and harder, on a typical medium/large solution there are lot's of servers at the request path: API Gateways, Load balancers, Cache servers, Proxies, and Firewalls, just to name a few. These generate latency and require much more engineering and monitoring to do it right.

The core idea of this project is to do more with less. pipehub being a programmable proxy, users can extend and customize it as needed. Features found in other servers can be added with Go packages instead of actual external services.

## How?
The code is extended with a thing called `handler`. It's a plain old Go code that is injected at compile time at the application. Being a Go project gives much higher flexibility at the handler because it can really be anything.

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
  path    = "github.com/pipehub/handler"
  version = "v0.5.1"
  alias   = "base"
}
```

The handler points to the place where the Go code is, it should be a `go gettable` project. A handler is a generic processor that can be used on multiple hosts. A host track the endpoint the proxy gonna listen, where the origin is, and which handler gonna be used to process the requests.

A real example of a handler can be found [here](https://github.com/pipehub/handler).

## How to run it?
First, create a config file:
```bash
cp cmd/pipehub/pipehub/pipehub.sample.hcl cmd/pipehub/pipehub/pipehub.hcl
# edit cmd/pipehub/pipehub/pipehub.hcl
```

Generate the binary:
```bash
make generate
```

Execute it:
```
./cmd/pipehub/pipehub start -c ./cmd/pipehub/pipehub.hcl
```