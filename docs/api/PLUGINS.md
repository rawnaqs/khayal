# Plugin Development Guide

> How to build plugins and clients for Khayal. Updated: 2026-03-17

## What Plugins Need

| Need | Solution |
|------|----------|
| API contract | [openapi.yaml](openapi.yaml) |
| API reference | [REFERENCE.md](REFERENCE.md) |
| Authentication | [AUTH.md](AUTH.md) |
| Tech stack | Your choice - any HTTP client |

## Quick Start

### 1. Get Token

```bash
kl init
kl config view  # Copy token
```

### 2. Make API Call

Pick your language:

```go
// Go
client := &http.Client{}
req, _ := http.NewRequest("GET", "http://localhost:1133/v1/health", nil)
req.Header.Set("X-Khayal-Token", "your-token")
resp, _ := client.Do(req)
```

```python
# Python
import requests
headers = {'X-Khayal-Token': 'your-token'}
requests.get('http://localhost:1133/v1/health', headers=headers)
```

```javascript
// JavaScript
fetch('http://localhost:1133/v1/health', {
  headers: { 'X-Khayal-Token': 'your-token' }
})
```

```swift
// Swift
let url = URL(string: "http://localhost:1133/v1/health")!
var request = URLRequest(url: url)
request.setValue("your-token", forHTTPHeaderField: "X-Khayal-Token")
```

```kotlin
// Kotlin
val url = URL("http://localhost:1133/v1/health")
val connection = url.openConnection() as HttpURLConnection
connection.setRequestProperty("X-Khayal-Token", "your-token")
```

## Platform-Specific Guides

### Mobile App (iOS/Android)

**Recommended:**
- Swift (iOS) / Kotlin (Android)
- Store token in Keychain / EncryptedSharedPreferences

**Example:**
```swift
// iOS with Keychain
import Security

func saveToken(_ token: String) {
    let data = token.data(using: .utf8)!
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: "khayal",
        kSecAttrAccount as String: "token",
        kSecValueData as String: data
    ]
    SecItemDelete(query as CFDictionary)
    SecItemAdd(query as CFDictionary, nil)
}
```

```kotlin
// Android with EncryptedSharedPreferences
val masterKey = MasterKey.Builder(context)
    .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
    .build()

val encryptedPrefs = EncryptedSharedPreferences.create(
    context,
    "khayal_prefs",
    masterKey,
    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
)
encryptedPrefs.edit().putString("token", token).apply()
```

### Browser Extension

**Recommended:**
- TypeScript/JavaScript
- Use Chrome Extensions API or Web Extension API

**Example:**
```typescript
// background.ts
chrome.runtime.onInstalled.addListener(() => {
  // Prompt user for token
  chrome.storage.local.set({ khayalConfigured: false });
});

function captureToKhayal(content: string) {
  chrome.storage.local.get(['token', 'host'], (result) => {
    fetch(`${result.host}/v1/capture`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Khayal-Token': result.token
      },
      body: JSON.stringify({ type: 'text', content })
    });
  });
}
```

### Raycast Extension

**Recommended:**
- Swift/TypeScript
- Use Raycast API

**Example:**
```swift
@Command()
private func capture() {
    TextField("Thought", text: $thought)
    .onSubmit {
        let url = URL(string: "\(host)/v1/capture")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(token, forHTTPHeaderField: "X-Khayal-Token")
        request.httpBody = try? JSONEncoder().encode(["type": "text", "content": thought])
        
        URLSession.shared.dataTask(with: request) { _, _, _ in
            // Success
        }
    }
}
```

### Desktop Widget (macOS/Windows)

**Recommended:**
- SwiftUI (macOS) / Electron / Tauri
- Use native system tray

## Generated Clients

Use OpenAPI spec to generate clients:

```bash
# Install OpenAPI Generator
npm install -g @openapitools/openapi-generator-cli

# Generate client
openapi-generator generate \
  -i docs/api/openapi.yaml \
  -g <language> \
  -o client/<language>
```

**Available generators:**
- `go` - Go
- `python` - Python
- `typescript-axios` - TypeScript
- `swift5` - Swift
- `kotlin` - Kotlin
- `java` - Java
- `csharp` - C#
- `ruby` - Ruby
- `php` - PHP

## Architecture Recommendations

### Capture Interface Philosophy

All capture interfaces should:
1. **Send HTTP only** - No processing logic in client
2. **Include token** - Every request needs `X-Khayal-Token`
3. **Handle async** - Text is sync, image/URL are async
4. **Poll for status** - Check `/queue/{id}` for progress

### Data Flow

```
Plugin UI → HTTP POST /v1/capture → Khayal Server
                                     ↓
                              Worker processes
                                     ↓
                              Vault writes note
```

### Error Handling

1. **Check response status**: `done` vs `processing`
2. **Poll for async**: Use `/queue/{id}` for image/URL
3. **Handle errors**: Show user-friendly messages
4. **Retry failed**: Use `/queue/{id}/retry`

## Testing Your Plugin

```bash
# Start local server
khayal start

# Test capture
curl -X POST http://localhost:1133/v1/capture \
  -H "Content-Type: application/json" \
  -H "X-Khayal-Token: your-token" \
  -d '{"type": "text", "content": "test"}'

# Check health
curl http://localhost:1133/v1/health \
  -H "X-Khayal-Token: your-token"
```

## Example Projects

See these for reference implementations:
- `cli/` - Official Go CLI
- `ui/react/` - Official React PWA

## Need Help?

1. Check [REFERENCE.md](REFERENCE.md) for API details
2. Check [AUTH.md](AUTH.md) for authentication
3. Open an issue at github.com/rawnaqs/khayal/issues
