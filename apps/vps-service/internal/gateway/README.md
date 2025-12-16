# Gateway Bidirectional Stream Architecture

## Overview

The VPS service maintains a persistent bidirectional stream connection to the VPS gateway. This architecture allows:
- Gateway to send requests to VPS service instances (e.g., FindVPSByLease)
- VPS service to respond over the same connection
- Multiple VPS service instances (beta, production, dev) connecting to the same gateway
- Easy extension for new request types

## Architecture

```
┌─────────────┐                    ┌──────────────┐
│ VPS Gateway │◄───RegisterGateway─┤ VPS Service  │
│  (Server)   │                    │   (Client)   │
│             │                    │              │
│  - Tracks   │  Request →         │  - Connects  │
│    streams  │                    │    on start  │
│  - Sends    │                    │  - Handles   │
│    requests │  ← Response        │    requests  │
│             │                    │              │
└─────────────┘                    └──────────────┘
```

## Components

### VPS Gateway (Receives Connections)

**Location:** `apps/vps-gateway/internal/server/service.go`

The gateway service:
1. Accepts incoming `RegisterGateway` streams from VPS services
2. Stores streams in `connectedStreams` map
3. When DHCP manager needs to resolve a lease, sends `FindVPSByLease` requests to ALL connected VPS instances
4. Returns first valid response

**Key Methods:**
- `RegisterGateway()` - Handles incoming connections
- `FindVPSByLease()` - Sends requests to all connected VPS instances
- Implements `dhcp.APIClient` interface

### VPS Service (Initiates Connection)

**Location:** `apps/vps-service/internal/gateway/client.go`

The gateway client:
1. Connects to gateway on startup
2. Maintains persistent bidirectional stream
3. Registers handlers for different request methods
4. Routes incoming requests to appropriate handlers

**Key Types:**
```go
type RequestHandler interface {
    HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error)
}
```

## Adding New Request Types

### 1. Define Proto Messages

In `packages/proto/proto/obiente/cloud/vps/v1/vps_service.proto`:

```protobuf
message NewMethodRequest {
  string some_field = 1;
}

message NewMethodResponse {
  string result = 1;
}

service VPSService {
  rpc NewMethod(NewMethodRequest) returns (NewMethodResponse);
}
```

### 2. Create Handler in VPS Service

Create `apps/vps-service/internal/gateway/handlers.go` or add to existing:

```go
type NewMethodHandler struct {
    // dependencies
}

func NewNewMethodHandler(deps) *NewMethodHandler {
    return &NewMethodHandler{...}
}

func (h *NewMethodHandler) HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error) {
    // 1. Unmarshal request
    var req vpsv1.NewMethodRequest
    if err := proto.Unmarshal(payload, &req); err != nil {
        return nil, fmt.Errorf("failed to unmarshal: %w", err)
    }

    // 2. Process request
    result := doSomething(req)

    // 3. Create and marshal response
    resp := &vpsv1.NewMethodResponse{Result: result}
    respPayload, err := proto.Marshal(resp)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal response: %w", err)
    }

    return respPayload, nil
}
```

### 3. Register Handler in main.go

In `apps/vps-service/main.go`:

```go
gatewayClient.RegisterHandler("NewMethod", gateway.NewNewMethodHandler(deps))
```

### 4. Call from Gateway

In gateway service (`apps/vps-gateway/internal/server/service.go`), create a method:

```go
func (s *GatewayService) CallNewMethod(ctx context.Context, someField string) (*vpsv1.NewMethodResponse, error) {
    // Marshal request
    req := &vpsv1.NewMethodRequest{SomeField: someField}
    payload, err := proto.Marshal(req)
    if err != nil {
        return nil, err
    }

    // Send to all connected VPS instances
    s.streamsMu.RLock()
    // ... similar pattern to FindVPSByLease
    s.streamsMu.RUnlock()

    // Return first valid response
}
```

## Environment Variables

**VPS Service:**
- `VPS_GATEWAY_URL` - Gateway URL (e.g., `http://vps-gateway:1537`)
- `VPS_GATEWAY_API_SECRET` - Shared secret for authentication

**VPS Gateway:**
- `GATEWAY_API_SECRET` - Must match VPS service secret
- `GATEWAY_GRPC_PORT` - Port to listen on (default: 1537)

## Message Flow Example

### FindVPSByLease Request

1. **Gateway** detects lease with non-VPS hostname
2. **Gateway** calls `gatewayService.FindVPSByLease(ip, mac)`
3. **Gateway** sends request to ALL connected VPS instances:
   ```
   GatewayMessage {
       Type: "request",
       Request: {
           RequestId: "gateway-findvps-123-prod",
           Method: "FindVPSByLease",
           Payload: marshaled(FindVPSByLeaseRequest)
       }
   }
   ```
4. **VPS Service** receives request, routes to FindVPSByLeaseHandler
5. **Handler** queries database, marshals response
6. **VPS Service** sends response:
   ```
   GatewayMessage {
       Type: "response",
       Response: {
           RequestId: "gateway-findvps-123-prod",
           Success: true,
           Payload: marshaled(FindVPSByLeaseResponse)
       }
   }
   ```
7. **Gateway** receives response, unmarshals, returns to DHCP manager
8. **DHCP Manager** updates hosts file with VPS ID

## Benefits

- ✅ **Extensible:** Add new methods without changing core infrastructure
- ✅ **Clean Separation:** Request handlers are independent, testable units
- ✅ **Type-Safe:** Proto definitions ensure contract compliance
- ✅ **Multi-Instance:** Gateway queries all VPS instances (beta, prod, dev)
- ✅ **Resilient:** Handles partial failures gracefully
- ✅ **Maintainable:** Clear patterns for adding functionality
