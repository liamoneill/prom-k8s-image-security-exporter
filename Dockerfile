FROM golang:1.16 as build

WORKDIR /go/src/app

RUN go get -u -v github.com/mgechev/revive

COPY go.mod go.sum /go/src/app/
RUN go mod download

COPY . /go/src/app/

ARG GIT_REVISION

RUN set -ex \
  && go build \
    -ldflags="-X main.gitRevision=${GIT_REVISION}" \
    -o k8s-image-exporter \
  && go test -v ./...
  # && revive \
  # && revive -config .revive.toml -formatter friendly


FROM centos:8

# hadolint ignore=DL3041
RUN set -ex \
  && dnf install -y skopeo \
  && dnf clean all

RUN useradd --system --shell /bin/false --home /app --user-group app
USER app
COPY --from=build --chown=app:app /go/src/app/k8s-image-exporter /app/bin/
ENTRYPOINT ["/app/bin/k8s-image-exporter"]
