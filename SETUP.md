# Les'Go Multi-Device Setup Guide

This guide explains how to connect two different laptops using Les'Go, either via the public relay server or a local network.

## Option 1: Using the Public Relay Server (Easiest)
Once the server is deployed to `lesgo.xplnhub.com`, no configuration is needed.

1.  **On both laptops**, install Les'Go:
    ```bash
    go install github.com/XplnHUB/Les-Go/client@latest
    ```
2.  **Laptop A**: Go online.
    ```bash
    lesgo
    ```
    *Note your Device ID (e.g., `1234567890`).*
3.  **Laptop B**: Connect to Laptop A.
    ```bash
    lesgo 1234567890
    ```

---

## Option 2: Using a Private Local Network (Local Setup)
If you are on the same Wi-Fi and want to run your own server:

### 1. Find the Server IP
On the laptop that will host the server (Laptop A), find its local IP address:
- **Mac/Linux**: `ipconfig getifaddr en0` or `ifconfig | grep "inet "`
- **Windows**: `ipconfig`
*Assume the IP is `192.168.1.5`.*

### 2. Start the Server (Laptop A)
```bash
# Run the server on port 80
sudo PORT=80 go run ./server/main.go
```

### 3. Connect the Clients
**On Both Laptop A and Laptop B**, point the client to the server's IP:

```bash
# Set the server address
export LESGO_SERVER=192.168.1.5:80

# Laptop A: Go online
lesgo online

# Laptop B: Connect to Laptop A
lesgo connect <LAPTOP_A_ID>
```

---

## Troubleshooting

### Connection Refused
- Ensure the server is running and the port is open in the firewall.
- If using port 80, you may need `sudo`. You can use `PORT=8080` instead if preferred.

### "Server Unavailable"
- Double check the `LESGO_SERVER` environment variable: `echo $LESGO_SERVER`.
- Try to `ping` the server laptop's IP from the other laptop.

### Encryption Error
- Ensure both laptops are running the exact same version of Les'Go.
