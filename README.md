# modware-user
[dictyBase](http://dictybase.org) **API** server that uses [dictycontent
backend](https://github.com/dictybase-docker/dictycontent-postgres) to manage
the data from rich text editor frontend. The API server supports both gRPC and
HTTP/JSON protocol for data exchange.

## Usage
```
NAME:
   modware-content - starts the modware-content microservice with HTTP and grpc backends

USAGE:
   modware-content [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --dictycontent-pass value  dictycontent database password [$DICTYCONTENT_PASS]
   --dictycontent-db value    dictycontent database name [$DICTYCONTENT_DB]
   --dictycontent-user value  dictycontent database user [$DICTYCONTENT_USER]
   --dictycontent-host value  dictycontent database host (default: "dictycontent-backend") [$DICTYCONTENT_BACKEND_SERVICE_HOST]
   --dictycontent-port value  dictycontent database port [$DICTYCONTENT_BACKEND_SERVICE_PORT]
   --port value               tcp port at which the servers will be available (default: "9597")
   --help, -h                 show help
   --version, -v              print the version

```
## API
#### [HTTP/JSON](https://dictybase.github.io/dictybase-api)
#### gRPC 
The protocol buffer definitions and service apis are documented
[here](https://github.com/dictyBase/dictybaseapis/tree/master/dictybase/content).

