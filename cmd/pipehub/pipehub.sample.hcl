core {
  graceful-shutdown = "10s"

  http {
    server {
      listen {
        port = 80
      }

      action {
        not-found = "base.NotFound"
        panic     = "base.Panic"
      }
    }
  }
}

http "google" {
  handler = "base.Default"
}

pipe "github.com/pipehub/sample" {
  version = "v0.9.0"
  alias   = "base"

  config {
    host = "https://www.google.com"
  }
}
