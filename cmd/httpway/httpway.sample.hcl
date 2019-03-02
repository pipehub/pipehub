handler {
  path    = "github.com/httpway/httpway-default"
  version = "v1.1.0"
  alias   = "base"
}

handler {
  path    = "github.com/httpway/httpway-businesslogic"
  version = "v2.1.3"
  alias   = "bl"
}

server {
  graceful-shutdown = "10s"

  http {
    port = 80
  }
}
