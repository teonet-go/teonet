module github.com/teonet-go/teonet/cmd/teonet

go 1.21.1

replace github.com/teonet-go/teonet => ../..

replace github.com/teonet-go/teonet/cmd/teonet/menu => ./menu

// replace github.com/teonet-go/teocrypt => ../../../teocrypt

require (
	github.com/chzyer/readline v1.5.1
	github.com/teonet-go/teocrypt v0.0.3
	github.com/teonet-go/teomon v0.5.14
	github.com/teonet-go/teonet v0.6.4
)

require (
	github.com/FactomProject/basen v0.0.0-20150613233007-fe3947df716e // indirect
	github.com/FactomProject/btcutilecc v0.0.0-20130527213604-d3a63a5752ec // indirect
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/kirill-scherba/bslice v0.0.2 // indirect
	github.com/kirill-scherba/stable v0.0.8 // indirect
	github.com/teonet-go/tru v0.0.18 // indirect
	github.com/tyler-smith/go-bip32 v1.0.0 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
)
