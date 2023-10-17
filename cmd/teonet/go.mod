module github.com/teonet-go/teonet/cmd/teonet

go 1.21.1

replace github.com/teonet-go/teonet => ../..

replace github.com/teonet-go/teonet/cmd/teonet/menu => ./menu

require (
	github.com/chzyer/readline v1.5.1
	github.com/teonet-go/teomon v0.5.14
	github.com/teonet-go/teonet v0.6.4
)

require (
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/kirill-scherba/bslice v0.0.2 // indirect
	github.com/kirill-scherba/stable v0.0.8 // indirect
	github.com/teonet-go/tru v0.0.18 // indirect
	golang.org/x/sys v0.13.0 // indirect
)