module posts

go 1.15

require (
	github.com/golang/protobuf v1.5.3
	github.com/gosimple/slug v1.9.0
	github.com/micro/dev v0.0.0-20201103105140-02e00085dfa7
	micro.dev/v4 v4.0.0-20230710120043-bdb1ee096b24
)

// This can be removed once etcd becomes go gettable, version 3.4 and 3.5 is not,
// see https://github.com/etcd-io/etcd/issues/11154 and https://github.com/etcd-io/etcd/issues/11931.
replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
