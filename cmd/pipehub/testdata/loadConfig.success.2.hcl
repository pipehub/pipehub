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

http "google" {
  handler = "base.Default"
}

pipe "github.com/pipehub/handler" {
  version = "v0.7.0"
  alias   = "base"
}
