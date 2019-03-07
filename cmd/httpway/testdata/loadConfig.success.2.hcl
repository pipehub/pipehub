server {
  graceful-shutdown = "10s"

  action {
    not-found = "base.NotFound"
    panic     = "base.Panic"
  }

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
  version = "v0.5.1"
  alias   = "base"
}
