# About

Use this repo to analyse if your wallet addresses have been reported in layerzero initial list or on their github issues. Github issue statue and labels are also logged with flagged wallets. If label is valid then layerzero accepted the reporting. A summary is also stored in an output file.

# How to use

- [install golang](https://go.dev/doc/install)
- rename `dev.env` to `.env` file
- add your github token [link](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) in env file
- add your wallets in a single column without any header or comma in a file. add the file path to env file
- run `go mod install` to install dependencies
- run `go run main.go`