# Bulk Operations

This guide demonstrates how to use the bulk operations API client for atomic record management.

**Note:** The bulk operations functionality is available through the Go client API but not yet exposed as a Terraform resource. This is intended for advanced use cases where you might extend the provider or use the client library directly in custom Go code.

## Usage

The bulk operations API allows you to perform multiple record operations atomically:
- If any operation fails, all operations are rolled back
- Supports create, update, and delete actions
- More efficient than individual API calls for large-scale changes

## Example Code

```go
package main

import (
    "context"
    "fmt"
    "github.com/poweradmin/terraform-provider-poweradmin/internal/provider"
)

func main() {
    // Create client
    config := &provider.PoweradminProviderModel{
        ApiUrl:     types.StringValue("http://localhost:3000"),
        Username:   types.StringValue("admin"),
        Password:   types.StringValue("poweradmin123"),
        ApiVersion: types.StringValue("v2"),
        Insecure:   types.BoolValue(true),
    }

    client, err := provider.NewClient(config)
    if err != nil {
        panic(err)
    }

    // Prepare bulk operations
    bulkReq := provider.BulkRecordsRequest{
        Operations: []provider.BulkRecordOperation{
            {
                Action:  "create",
                Name:    "www1",
                Type:    "A",
                Content: "192.0.2.10",
                TTL:     3600,
            },
            {
                Action:  "create",
                Name:    "www2",
                Type:    "A",
                Content: "192.0.2.11",
                TTL:     3600,
            },
            {
                Action:   "create",
                Name:     "mail",
                Type:     "MX",
                Content:  "mail.example.com",
                TTL:      3600,
                Priority: 10,
            },
        },
    }

    // Execute bulk operations
    zoneID := int64(1)
    result, err := client.BulkRecordOperations(context.Background(), zoneID, bulkReq)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Success: %d operations\n", result.SuccessCount)
    fmt.Printf("Failed: %d operations\n", result.FailureCount)
    if len(result.Errors) > 0 {
        fmt.Printf("Errors: %v\n", result.Errors)
    }
}
```

## Features

- **Atomic Operations**: All operations succeed or all fail (no partial updates)
- **Priority Support**: Full support for MX, SRV priority fields
- **Error Reporting**: Detailed error messages for failed operations
- **Performance**: More efficient than individual API calls
