# Khayal Authentication Guide

> How to authenticate with the Khayal API. Updated: 2026-03-17

## Overview

Khayal uses a simple token-based authentication. Every request must include a valid token in the `X-Khayal-Token` header.

## Token

- **Format**: 32-byte hex string
- **Generation**: Auto-generated on first run
- **Storage**: `~/.config/khayal/config.yaml`
- **CLI config**: `~/.config/khayal/kl.yaml`

## Using the Token

### HTTP Header

```bash
curl -H "X-Khayal-Token: a1b2c3d4e5f6..." \
     http://localhost:1133/v1/health
```

### In Code

```go
client := &http.Client{}
req, _ := http.NewRequest("GET", "http://localhost:1133/v1/health", nil)
req.Header.Set("X-Khayal-Token", "a1b2c3d4e5f6...")
resp, _ := client.Do(req)
```

### JavaScript/Fetch

```javascript
fetch('http://localhost:1133/v1/health', {
  headers: {
    'X-Khayal-Token': 'a1b2c3d4e5f6...'
  }
})
```

### Python

```python
import requests

headers = {'X-Khayal-Token': 'a1b2c3d4e5f6...'}
response = requests.get('http://localhost:1133/v1/health', headers=headers)
```

## Getting Your Token

### Via CLI

```bash
kl config view
# or
cat ~/.config/khayal/config.yaml | grep token
```

### First Run

```bash
$ khayal init
Created ~/.config/khayal/config.yaml
Generated token: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6
```

**Important**: The token is shown only once. Save it securely.

## Regenerating Token

### Via CLI

```bash
kl config set token <new-token>
```

### Manually

Edit `~/.config/khayal/config.yaml`:

```yaml
server:
  token: your-new-token-here
```

## Token Security

| Rule | Reason |
|------|--------|
| Never log token | Security requirement |
| Store in config with 600 permissions | Prevent unauthorized access |
| Don't commit to git | Prevent accidental exposure |
| Use different token per device | Limit exposure if compromised |

## For Plugins

### Mobile Apps

Store token securely:
- **iOS**: Keychain
- **Android**: EncryptedSharedPreferences

```swift
// iOS - Keychain
let token = Keychain.load("khayal_token")

// Android - EncryptedSharedPreferences
val token = encryptedPrefs.getString("khayal_token", null)
```

### Browser Extensions

- Use `chrome.storage` with encryption
- Or prompt user for token on install

### Example: React Native

```typescript
import * as Keychain from 'react-native-keychain';

const getToken = async () => {
  const credentials = await Keychain.getGenericPassword({ service: 'khayal' });
  return credentials.password;
};

const apiCall = async (endpoint: string) => {
  const token = await getToken();
  return fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'X-Khayal-Token': token!,
    },
  });
};
```

## Server Configuration

### Default Binding

```yaml
server:
  host: 127.0.0.1  # Never 0.0.0.0 by default
  port: 1133
```

### For Remote Access

```yaml
server:
  host: 0.0.0.0  # Listen on all interfaces
  port: 1133
```

**Warning**: Only do this on trusted networks or with VPN (e.g., Tailscale).

## Troubleshooting

### 401 Unauthorized

```json
{ "error": "invalid token", "code": "AUTH_001" }
```

**Solutions:**
1. Check token is correct
2. Check token is in header: `X-Khayal-Token`
3. Check token matches config: `kl config view`

### Token Not Working

```bash
# Regenerate token
kl config set token <new-hex-token>

# Restart server
khayal start
```

## Security Best Practices

1. **Local only**: Default bind to `127.0.0.1`
2. **Token-only**: No session tokens, no OAuth
3. **No token in logs**: Server never logs tokens
4. **Permissions**: Config file at 600
5. **Network**: Use VPN for remote access

## Environment Variables (Alternative)

For CI/CD or scripts:

```bash
export KHAYAL_TOKEN="your-token"
export KHAYAL_HOST="http://localhost:1133"

# Use in scripts
curl -H "X-Khayal-Token: $KHAYAL_TOKEN" \
     "$KHAYAL_HOST/v1/health"
```
