module github.com/levitateos/sodaos

go 1.26.7

require (
	forgejo.org/extension-sdk v0.0.0
	github.com/coder/websocket v1.8.15
	github.com/dicebear/dicebear-go/v10 v10.7.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/crypto v0.55.0
	golang.org/x/oauth2 v0.34.0
	golang.org/x/sys v0.47.0
	modernc.org/sqlite v1.58.0
	tailscale.com/client/tailscale/v2 v2.10.1
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/dicebear/schema v1.5.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.8.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/kisielk/errcheck v1.9.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/tailscale/hujson v0.0.0-20220506213045-af5ed07155e5 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.61.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	honnef.co/go/tools v0.8.1 // indirect
	modernc.org/libc v1.75.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	mvdan.cc/gofumpt v0.9.1 // indirect
)

tool (
	github.com/fzipp/gocyclo/cmd/gocyclo
	github.com/kisielk/errcheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)

replace forgejo.org/extension-sdk => ../forgejo-ext/sdk
