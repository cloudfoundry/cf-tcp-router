# cf-tcp-router

TCP Router repository for Cloud Foundry

**Note**: This repository should be imported as `code.cloudfoundry.org/cf-tcp-router`.

Subscribes to SSE events from
[routing-api](https://github.com/cloudfoundry/routing-api) to update haproxy
configuration for configuring tcp routes.

## Development

### <a name="dependencies"></a>Dependencies

This repository's dependencies are managed using
[routing-release](https://github.com/cloudfoundry/routing-release). Please refer to documentation in that repository for setting up tests

### Executables

1. `bin/test.bash`: This file is used to run test in Docker & CI. Please refer to [Dependencies](#dependencies) for setting up tests.

### Reporting issues and requesting features

Please report all issues and feature requests in [cloudfoundry/routing-release](https://github.com/cloudfoundry/routing-release).
