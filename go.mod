module github.com/yindaheng98/dion

go 1.16

require (
	github.com/bep/debounce v1.2.0 // indirect
	github.com/cloudwebrtc/nats-discovery v0.3.0
	github.com/cloudwebrtc/nats-grpc v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/pion/ion v1.10.0
	github.com/pion/ion-log v1.2.2
	github.com/pion/ion-sdk-go v0.7.0
	github.com/pion/ion-sfu v1.11.0
	github.com/pion/webrtc/v3 v3.1.7 // indirect
	github.com/spf13/viper v1.11.0
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
)

replace (
	github.com/pion/ion v1.10.0 => github.com/yindaheng98/ion v1.10.1-0.20220426115625-f36c15970110
	github.com/pion/ion-sdk-go v0.7.0 => github.com/yindaheng98/ion-sdk-go v0.7.1-0.20220426113245-a9894b608a13
)
