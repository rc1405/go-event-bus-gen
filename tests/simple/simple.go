package simple

//go:generate go-event-bus-gen --in simple.proto --out bus.go
//go:generate mockgen -source=bus.go -destination mocks.go -package simple
