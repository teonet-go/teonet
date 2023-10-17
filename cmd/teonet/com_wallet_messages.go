package main

var (
	// Wallet commands usage ---------------------------------------------------
	usageCommand = `
usage: api -wallet <address> <command> [arguments...]

Application wallet commands:
  new
        creates new wallet mnemonic (create new wallet)
  insert <mnemonic>
        inserts your previously saved wallet mnemonic (import wallet)
  show
        shows current wallet mnemonic and private key (export wallet)
  save
        save current wallet parameters to this host (save wallet)
  load
        load saved wallet parameters from this host (load wallet)
  delete
        delete saved wallet parameters from this host (delete wallet)
  password <password>
        sets password to save and read mnemonic and master key at this host`

	// New command message -----------------------------------------------------
	descriptionNew = `New wallet for app %s created.

To show created wallet mnemonic - execute 'show' command:

` + color("api -wallet teos3 show") + `

To save created wallet mnemonic on this host - execute 'save' command:

` + color("api -wallet teos3 save") + `

** If you used another wallet before this command and it mnemonic was not saved
than you lost it. If your previously wallet was saved on this host but you have
not copy it than execute 'load' command:

` + color("api -wallet teos3 load") + `
`

	// Show message ------------------------------------------------------------
	descriptionShow = `Wallet for app %s codes:

mnemonic:
%s

private key:
%s
`

	// Show error message ------------------------------------------------------
	descriptionShowError = `Wallet for %s was not created.

To create new wallent use next command:

` + color("api -wallet teos3 new") + `

To load saved wallent use next command:

` + color("api -wallet teos3 load") + `
`

	// Load message ------------------------------------------------------------
	descriptionLoad = `Wallet for app %s loaded.

To show loaded wallet mnemonic - execute 'show' command:

` + color("api -wallet teos3 show") + `
`
)
