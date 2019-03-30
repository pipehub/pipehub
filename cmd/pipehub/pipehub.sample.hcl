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

    client {
      disable-keep-alive      = false
      disable-compression     = false
      max-idle-conns          = 1000
      max-idle-conns-per-host = 100
      max-conns-per-host      = 1000
      idle-conn-timeout       = "90s"
      tls-handshake-timeout   = "10s"
      expect-continue-timeout = "1s"
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
