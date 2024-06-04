## sei-sdk: Interact with Sei from Golang

The sei-sdk library provides a user-friendly Go interface for interacting with the Sei blockchain. It simplifies sending transactions, querying data, and managing signers for your applications.

### Features

* **Easy Interaction:** Interact with Sei blockchain through a high-level Go API.
* **Bank Operations:** Query account balances and potentially interact with the bank module in the future.
* **Wasm Support:** Interact with Wasm contracts on Sei, including sending and executing messages, as well as contract instantiation.
* **Transaction Management:** Sign and broadcast transactions to the Sei network.
* **Signer Management:** Add and manage signers for your application.
* **Transaction Retrieval:** Retrieve transaction details by their hash.

### Installation

```bash
go get github.com/spell-club/sei-sdk
```

### Usage

**1. Importing the Package**

```go
import (
  "context"
  "github.com/spell-club/sei-sdk"
)
```

**2. Creating a Client**

Before using any functionality, you need to create a `sei.Client` instance. You can do this by providing configuration details:

```go
cfg := sei.Config{
  RPCHost: "http://localhost:26657", // Replace with your Sei node RPC address
  ChainID: ChainIDTestnet,           // Replace with your Sei chain ID
}

client, err := sei.NewClient(cfg)
if err != nil {
  // Handle error
}
```

**3. Interacting with Sei**

The `sei.Client` provides various methods for interacting with the Sei blockchain. Here's a breakdown of some core functionalities:

**3.1 Querying Bank Balances**

```go
address := "sei1qspfj..." // Replace with the address you want to query
denom := "usei"        // Replace with the desired denomination (e.g., usei)

balance, err := client.GetBankBalance(context.Background(), address, denom)
if err != nil {
  // Handle error
}

fmt.Println("Balance:", balance.Amount.String())
```

**3.2 Interacting with Wasm Contracts**

**3.2.1 Sending Arbitrary JSON Messages (ExecuteJson):**

```go
contractAddress := "sei1...” // Replace with the contract address
signerName := "my-signer"   // Replace with the signer name you added

// Define your message data as a Go struct or map
var msgData = struct {
  Action string `json:"action"`
  Data   string `json:"data"`
}{
  Action: "deposit",
  Data:   "10usei",
}

resp, err := client.ExecuteJson(context.Background(), signerName, contractAddress, msgData)
if err != nil {
  // Handle error
}

fmt.Println("Transaction Hash:", resp.TxHash)
```

**3.2.2 Sending Arbitrary JSON Messages for Contract Instantiation (InstantiateJson):**

```go
codeID := uint64(123)       // Replace with the Wasm code ID for your contract
label := "my-contract"     // Replace with a unique label for your contract

// Define your instantiation message data as a Go struct or map
var instantiateMsg = struct {
  Name string `json:"name"`
  ...   // Other fields
}{
  Name: "My Wasm Contract",
}

funds := []sdktypes.Coin{
  // Define coins to send for contract instantiation (optional)
}

resp, err := client.InstantiateJson(context.Background(), signerName, codeID, label, instantiateMsg, funds)
if err != nil {
  // Handle error
}

fmt.Println("Transaction Hash:", resp.TxHash)
```

**3.3 Managing Signers**

Before interacting with the blockchain and signing transactions, you need to add signers to your `sei.Client` instance:

```go
address, err := client.AddSigner("my-signer", "your-mnemonic")
if err != nil {
  // Handle error (e.g., invalid mnemonic)
}
```

**3.4 Retrieving Transactions**

```go
txHash := "FABCDE..." // Replace with the transaction hash you want to retrieve

txResp, err := client.GetTxByHash(context.Background(), txHash, 3, 5*time.Second)
if err != nil {
  // Handle error
}

fmt.Println("Transaction result:", txResp.TxResult)
```
