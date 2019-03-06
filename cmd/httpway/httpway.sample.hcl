server {
  graceful-shutdown = "10s"

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
  version = "v0.3.0"
  alias   = "base"
}
