# payment discovery
The command line interface for bitmark Discovery proxy.

## Installation

```
$ go get github.com/bitmark-inc/discovery
$ go install github.com/bitmark-inc/discovery/commands/payment-discovery
```

## Usage

### Subscribe payment transactions

```
payment-discovery listen -addr <discovery_pub_endpoint> -currency <btc|ltc>
```

### Retrieve payment transactions after the specified timestamp

```
payment-discovery query -addr <discovery_rep_endpoint> -currency <btc|ltc> -timestamp <epoch_time_in_seconds>
```
