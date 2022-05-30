module github.com/yindaheng98/dion

go 1.18

require (
	github.com/bep/debounce v1.2.0
	github.com/cloudwebrtc/nats-discovery v0.3.0
	github.com/cloudwebrtc/nats-grpc v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/mitchellh/mapstructure v1.4.3
	github.com/nats-io/nats.go v1.12.0
	github.com/pelletier/go-toml v1.9.4
	github.com/pion/interceptor v0.1.10
	github.com/pion/ion v1.10.0
	github.com/pion/ion-log v1.2.2
	github.com/pion/ion-sdk-go v0.7.0
	github.com/pion/ion-sfu v1.11.0
	github.com/pion/sdp/v3 v3.0.4
	github.com/pion/webrtc/v3 v3.1.25
	github.com/spf13/viper v1.11.0
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/ebml-go/ebml v0.0.0-20160925193348-ca8851a10894 // indirect
	github.com/ebml-go/webm v0.0.0-20160924163542-629e38feef2a // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/fullstorydev/grpcurl v1.8.0 // indirect
	github.com/gammazero/deque v0.1.0 // indirect
	github.com/gammazero/workerpool v1.1.2 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/go-logr/zerologr v1.2.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/improbable-eng/grpc-web v0.14.1 // indirect
	github.com/jhump/protoreflect v1.8.2 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/lucsky/cuid v1.2.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.0.0-beta.8 // indirect
	github.com/petar/GoLLRB v0.0.0-20210522233825-ae3b015fd3e9 // indirect
	github.com/pion/datachannel v1.5.2 // indirect
	github.com/pion/dtls/v2 v2.1.3 // indirect
	github.com/pion/ice/v2 v2.2.2 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.5 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.9 // indirect
	github.com/pion/rtp v1.7.7 // indirect
	github.com/pion/sctp v1.8.2 // indirect
	github.com/pion/srtp/v2 v2.0.5 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pion/transport v0.13.0 // indirect
	github.com/pion/turn/v2 v2.0.8 // indirect
	github.com/pion/udp v0.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/rs/zerolog v1.26.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/net v0.0.0-20220412020605-290c469a71a5 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	nhooyr.io/websocket v1.8.6 // indirect
)

replace (
	github.com/pion/ion v1.10.0 => github.com/yindaheng98/ion v1.10.1-0.20220518115802-da154fb3ee21
	github.com/pion/ion-sfu v1.11.0 => github.com/yindaheng98/ion-sfu v1.11.1-0.20220521131211-10a33cc613d5
)
