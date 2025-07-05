# tedge-oscar

A Go CLI tool to manage thin-edge.io flows, including pulling and pushing flow images as OCI artifacts using the oras library, and managing flow instances.

## Commands

- `tedge-oscar flows images pull` — Pull a flow image from an OCI registry
- `tedge-oscar flows images push` — Push a flow image to an OCI registry
- `tedge-oscar flows images list` — List available flow images
- `tedge-oscar flows instances list` — List deployed flow instances
- `tedge-oscar flows instances deploy` — Deploy a flow instance

## Development

- Built with [Cobra](https://github.com/spf13/cobra) for CLI structure
- Uses [oras](https://github.com/oras-project/oras-go) for OCI artifact operations

## Getting Started

1. Install Go 1.21 or newer
2. Run `go mod tidy` to install dependencies
3. Build: `go build`
4. Run: `./tedge-oscar [command]`

## License

Apache 2.0
