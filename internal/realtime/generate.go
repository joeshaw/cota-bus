package realtime

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs-realtime.proto=github.com/joeshaw/cota-bus/internal/realtime gtfs-realtime.proto
