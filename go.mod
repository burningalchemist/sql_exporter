module github.com/burningalchemist/sql_exporter

go 1.18

require (
	github.com/SAP/go-ase v0.0.0-20230328095936-414598c937a8
	github.com/kardianos/minwinsvc v1.0.2
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/client_model v0.3.0
	github.com/prometheus/common v0.41.0
	github.com/prometheus/exporter-toolkit v0.9.1
	github.com/xo/dburl v0.14.2
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/klog/v2 v2.70.1
)

require (
	github.com/SAP/go-dblib v0.0.0-20220825075032-c1f3f4d6e7b3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace k8s.io/klog/v2 => github.com/simonpasquier/klog-gokit/v3 v3.1.0

replace github.com/xo/dburl => github.com/burningalchemist/dburl v0.0.0-20230420152839-10ad3d07a913
