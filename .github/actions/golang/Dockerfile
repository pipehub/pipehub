FROM golang:1.12.0-stretch

LABEL "name"="Golang"
LABEL "maintainer"="Diego Bernardes <diego.bernardes@outlook.com>"
LABEL "version"="1.0.0"

LABEL "com.github.actions.icon"="code"
LABEL "com.github.actions.color"="green-dark"
LABEL "com.github.actions.name"="Go"
LABEL "com.github.actions.description"="This is a Action to run Go tools and commands."

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
