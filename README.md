# modware-user

[![License](https://img.shields.io/badge/License-BSD%202--Clause-blue.svg)](LICENSE)  
![GitHub action](https://github.com/dictyBase/modware-content/workflows/Continuous%integration/badge.svg)
[![codecov](https://codecov.io/gh/dictyBase/modware-content/branch/develop/graph/badge.svg)](https://codecov.io/gh/dictyBase/modware-content)  
[![Technical debt](https://badgen.net/codeclimate/tech-debt/dictyBase/modware-content)](https://codeclimate.com/github/dictyBase/modware-content/trends/technical_debt)
[![Issues](https://badgen.net/codeclimate/issues/dictyBase/modware-content)](https://codeclimate.com/github/dictyBase/modware-content/issues)
[![Maintainability](https://api.codeclimate.com/v1/badges/21ed283a6186cfa3d003/maintainability)](https://codeclimate.com/github/dictyBase/modware-content/maintainability)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=dictyBase/modware-content)](https://dependabot.com)  
![Issues](https://badgen.net/github/issues/dictyBase/modware-content)
![Open Issues](https://badgen.net/github/open-issues/dictyBase/modware-content)
![Closed Issues](https://badgen.net/github/closed-issues/dictyBase/modware-content)  
![Total PRS](https://badgen.net/github/prs/dictyBase/modware-content)
![Open PRS](https://badgen.net/github/open-prs/dictyBase/modware-content)
![Closed PRS](https://badgen.net/github/closed-prs/dictyBase/modware-content)
![Merged PRS](https://badgen.net/github/merged-prs/dictyBase/modware-content)  
![Commits](https://badgen.net/github/commits/dictyBase/modware-content/develop)
![Last commit](https://badgen.net/github/last-commit/dictyBase/modware-content/develop)
![Branches](https://badgen.net/github/branches/dictyBase/modware-content)
![Tags](https://badgen.net/github/tags/dictyBase/modware-content/?color=cyan)  
![GitHub repo size](https://img.shields.io/github/repo-size/dictyBase/modware-content?style=plastic)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/dictyBase/modware-content?style=plastic)
[![Lines of Code](https://badgen.net/codeclimate/loc/dictyBase/modware-content)](https://codeclimate.com/github/dictyBase/modware-content/code)  
[![Funding](https://badgen.net/badge/NIGMS/Rex%20L%20Chisholm,dictyBase/yellow?list=|)](https://projectreporter.nih.gov/project_info_description.cfm?aid=9476993)
[![Funding](https://badgen.net/badge/NIGMS/Rex%20L%20Chisholm,DSC/yellow?list=|)](https://projectreporter.nih.gov/project_info_description.cfm?aid=9438930)

[dictyBase](http://dictybase.org) **API** server that uses [dictycontent
backend](https://github.com/dictybase-docker/dictycontent-postgres) to manage
data from rich text editor frontend. The API server supports both gRPC and
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

#### HTTP/JSON

It's [here](https://dictybase.github.io/dictybase-api), make sure you use the content from the dropdown on the top right.

#### gRPC

The protocol buffer definitions and service apis are documented
[here](https://github.com/dictyBase/dictybaseapis/tree/master/dictybase/content).
