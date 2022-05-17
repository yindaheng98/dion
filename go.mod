module github.com/yindaheng98/dion

go 1.16

require (
	github.com/bep/debounce v1.2.0 // indirect
	github.com/cloudwebrtc/nats-discovery v0.3.0
	github.com/cloudwebrtc/nats-grpc v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
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

replace (
	github.com/pion/ion v1.10.0 => github.com/yindaheng98/ion v1.10.1-0.20220516141641-3d928e40847e
	github.com/pion/ion-sdk-go v0.7.0 => github.com/yindaheng98/ion-sdk-go v0.7.1-0.20220426113245-a9894b608a13
	github.com/pion/ion-sfu v1.11.0 => github.com/yindaheng98/ion-sfu v1.11.1-0.20220511114506-edb96d67ed96
)
