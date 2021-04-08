# Klever Challenge
API to provide the user an interface to upvote or downvote a known list of the main Cryptocurrencies.

## Usage
Add your application configuration to your `.env` file in the root of your project like example below:

```shell
DB_NAME=klever
DB_COLLECTION=cryptos
DB_HOST=localhost
DB_PORT=27017

API_PORT=50051
```

Go to root of your project and run `go run server/main.go` on your terminal.

It's done! API is running.

## Observation
For this example I use `evans` gRPC client. If you have this client installed, so run `evans -r repl` on your second terminal.