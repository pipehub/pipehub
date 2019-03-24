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

pipe "github.com/pipehub/sample" {
  version = "v0.9.0"
  alias   = "base"

  config {
    endpoint {
      "google" = "google.com"
    }
  }
}
