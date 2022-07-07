# Operand Go SDK

The official Go SDK for the [Operand](https://operand.ai) API. You can get your free API key by logging into the dashboard on our website, and navigating to Settings -> API Keys. If you have any questions, comments or feedback, please [reach out](mailto:morgan@operand.ai) (we usually don't bite)!

### Installation

To install the SDK, simply add the package using Go modules. Note: We currently support Go 1.18+.

```
go get -u github.com/operandinc/go-sdk
```

### Usage

As an example of the API in action, we're going to index some text documents and perform semantic search over them. This is incredibly easy with our Go SDK, or the [TypeScript SDK](https://github.com/operandinc/typescript-sdk).

For a full, working example of this SDK in action, please see [operand\_test.go](operand_test.go).

To run this test, you can do the following:

```
OPERAND_API_KEY=<your api key> go test
```
