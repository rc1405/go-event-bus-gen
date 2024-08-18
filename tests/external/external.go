package external

//go:generate go-event-bus-gen --in external.proto --out bus.go --config config.yaml
//go:generate mockgen -source=bus.go -destination mocks.go -package external
