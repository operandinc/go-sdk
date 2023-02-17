# Operand Go SDK

The official Go SDK for the Operand API. Read our API documentation here: [https://docs.operand.ai](https://docs.operand.ai). If you have any questions or feedback, please feel free to reach out to us [via email](mailto:support@operand.ai) or [join our Discord](https://operand.ai/discord).

### Installation

```bash
go get -u github.com/operandinc/go-sdk
```

### Usage

```go
client := operand.NewClient(os.Getenv("OPERAND_API_KEY"))
if _, err := client.CreateFile(
    context.Background(),
    "go.txt",
    nil,
    bytes.NewReader([]byte("Hello, World!")),
    nil,
); err != nil {
    // handle the error
}
```
