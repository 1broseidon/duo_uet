# Configuration Encryption

## 🔐 Problem: API Keys Stored in Plaintext

Currently, `config.yaml` stores Duo API credentials in plaintext:

```yaml
tenants:
  - admin_api_secret: "abc123..."  # ⚠️ PLAINTEXT
applications:
  - client_secret: "xyz789..."     # ⚠️ PLAINTEXT
```

**Security Risks:**
- Anyone with filesystem access can read credentials
- Accidental commits to public repos expose secrets
- Backups contain plaintext credentials
- Log files might capture secrets

---

## 🎯 Solution: Field-Level Encryption (Recommended)

Encrypt only sensitive fields while keeping structure visible.

### What It Looks Like

```yaml
# config.yaml (with encrypted secrets)
tenants:
  - id: "prod-tenant"
    name: "Production"                                    # ✅ VISIBLE
    admin_api_key: "DIxxxxxxxxxxxxxxxxxx"                 # ✅ VISIBLE
    admin_api_secret: "ENC[AES256_GCM,dGVzdA==,yNq7...]"  # 🔒 ENCRYPTED
    api_hostname: "api-12345678.duosecurity.com"          # ✅ VISIBLE

applications:
  - id: "websdk-prod"
    name: "Production WebSDK"                             # ✅ VISIBLE
    type: "websdk"                                        # ✅ VISIBLE
    enabled: true                                         # ✅ VISIBLE
    client_id: "DIxxxxxxxxxxxxxxxxxx"                     # ✅ VISIBLE
    client_secret: "ENC[AES256_GCM,YWJjZA==,Xp3lQ8...]"  # 🔒 ENCRYPTED
    api_hostname: "api-12345678.duosecurity.com"          # ✅ VISIBLE
```

### Benefits

✅ **Human-readable structure** - Can see app names, types, IDs
✅ **Easy debugging** - Know which application is which
✅ **Git-friendly** - Diffs show what changed (except secrets)
✅ **Selective encryption** - Only secrets are encrypted
✅ **Backward compatible** - Plaintext still works during migration

---

## 🔑 Key Management Options

### Option 1: Auto-Generated Key (Simplest)

On first run, generates `.uet_key` file:

```bash
$ ./uet
# Generates .uet_key with 256-bit random key
# File permissions: 0600 (owner read/write only)
```

**Pros:** Zero configuration
**Cons:** Key stored on same filesystem as config

### Option 2: Environment Variable (Production)

```bash
export UET_MASTER_KEY="your-secure-password-here"
./uet
```

**Pros:** Key separate from filesystem
**Cons:** Need to manage env var

### Option 3: External Key Management

```bash
# AWS Secrets Manager
export UET_MASTER_KEY=$(aws secretsmanager get-secret-value --secret-id uet-key --query SecretString --output text)

# HashiCorp Vault
export UET_MASTER_KEY=$(vault kv get -field=key secret/uet)

# 1Password CLI
export UET_MASTER_KEY=$(op read "op://Private/UET Master Key/password")
```

**Pros:** Enterprise-grade security
**Cons:** Requires additional infrastructure

---

## 📖 Usage

### Encrypting Existing Config

```go
package main

import (
    "log"
    "user_experience_toolkit/internal/config"
    "user_experience_toolkit/internal/crypto"
)

func main() {
    // Load existing config
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Create crypto manager (uses UET_MASTER_KEY or .uet_key)
    cm, err := crypto.NewCryptoManager()
    if err != nil {
        log.Fatal(err)
    }

    // Encrypt config
    if err := cfg.EncryptSecrets(cm); err != nil {
        log.Fatal(err)
    }

    // Save encrypted config
    if err := cfg.SaveConfig(); err != nil {
        log.Fatal(err)
    }

    log.Println("Config encrypted successfully!")
}
```

### Automatic Transparent Encryption

Once integrated into the config package, encryption/decryption happens automatically:

```go
// Load config (automatically decrypts secrets)
cfg, err := config.LoadConfig("config.yaml")

// Use plaintext values
app := cfg.Applications[0]
client := duo.NewClient(app.ClientSecret)  // Transparently decrypted

// Save config (automatically encrypts secrets)
cfg.SaveConfig()  // Secrets are encrypted before writing
```

---

## 🔒 Encryption Details

### Algorithm
- **Cipher:** AES-256-GCM (Galois/Counter Mode)
- **Key Derivation:** PBKDF2-SHA256 (100,000 iterations)
- **Nonce:** 12 bytes (random per encryption)
- **Authentication:** Built-in AEAD (prevents tampering)

### Security Properties
✅ **Confidentiality** - Secrets are unreadable without key
✅ **Integrity** - Tampering detected via authentication tag
✅ **Non-deterministic** - Same plaintext produces different ciphertext
✅ **Industry standard** - AES-256-GCM used by TLS 1.3, IPSec, etc.

### Format
```
ENC[AES256_GCM,<nonce>,<ciphertext>]
       │          │          └─ Base64-encoded ciphertext + auth tag
       │          └─ Base64-encoded 12-byte nonce
       └─ Algorithm identifier
```

---

## 🆚 Alternative Approaches

### Approach 1: Whole-File Encryption

Encrypt entire `config.yaml` file:

```bash
# Encrypted file (binary blob)
$ cat config.yaml
�K�2��m��x��...

# Decrypt to view/edit
$ age --decrypt -i ~/.age/key.txt config.yaml.age > config.yaml
$ vim config.yaml
$ age --encrypt -r age1abc... config.yaml > config.yaml.age
```

**Pros:**
- Simple implementation
- Standard tools (age, gpg)

**Cons:**
- ❌ Can't inspect structure without decrypting
- ❌ Git diffs are meaningless (binary changes)
- ❌ Can't selectively share config
- ❌ Have to decrypt entire file to change one value

### Approach 2: External Secrets Store

Store only references in `config.yaml`:

```yaml
applications:
  - client_secret: "{{vault:secret/duo/client_secret}}"
```

**Pros:**
- Centralized secret management
- Audit logs
- Rotation policies

**Cons:**
- ❌ Requires external infrastructure (Vault, AWS Secrets Manager)
- ❌ Network dependency
- ❌ Overkill for single-instance deployments
- ❌ More complex setup

### Approach 3: Encrypted Database

Use BadgerDB with encryption-at-rest:

**Pros:**
- All data encrypted
- Atomic operations

**Cons:**
- ❌ Loses human-readability completely
- ❌ Binary format (can't git diff)
- ❌ Overkill for this use case

---

## 📊 Comparison Matrix

| Feature | Plaintext YAML | Field-Level Encryption | Whole-File Encryption | External Secrets | Database |
|---------|----------------|------------------------|----------------------|------------------|----------|
| **Security** | ❌ Low | ✅ High | ✅ High | ✅ Highest | ✅ High |
| **Readability** | ✅ Full | ✅ Partial | ❌ None | ✅ Partial | ❌ None |
| **Git Diff** | ✅ Clear | ✅ Mostly | ❌ Binary | ✅ Mostly | ❌ Binary |
| **Debugging** | ✅ Easy | ✅ Easy | ❌ Hard | ✅ Easy | ❌ Hard |
| **Setup** | ✅ None | ✅ Minimal | ⚠️ Moderate | ❌ Complex | ⚠️ Moderate |
| **Dependencies** | ✅ None | ✅ Stdlib | ⚠️ External Tool | ❌ Infrastructure | ⚠️ BadgerDB |
| **Backup** | ✅ Easy | ✅ Easy | ⚠️ Need Key | ⚠️ Need Key | ⚠️ Export Tool |
| **Sharing** | ✅ Easy | ✅ Easy | ❌ Hard | ❌ Hard | ❌ Hard |

---

## 🎯 Recommendation

**Use Field-Level Encryption** for this project because:

1. **Perfect security/usability balance** - Secrets encrypted, structure visible
2. **Minimal code changes** - Transparent to rest of application
3. **Developer-friendly** - Easy to debug, inspect, share
4. **Git-friendly** - Can see what changed
5. **No external dependencies** - Pure Go stdlib
6. **Backward compatible** - Gradual migration path

---

## 🚀 Implementation Steps

### Phase 1: Add Crypto Package (Done ✅)
- [x] Create `internal/crypto/crypto.go`
- [x] Implement AES-256-GCM encryption
- [x] Add comprehensive tests
- [x] Support multiple key sources

### Phase 2: Integrate with Config
```go
// internal/config/config.go

// Add crypto manager to Config struct
type Config struct {
    // ... existing fields ...
    cryptoManager *crypto.CryptoManager
}

// LoadConfig with automatic decryption
func LoadConfig(path string) (*Config, error) {
    // Load YAML
    data, err := os.ReadFile(path)
    // ... parse YAML ...

    // Initialize crypto manager
    cm, err := crypto.NewCryptoManager()

    // Decrypt secrets
    for i := range cfg.Tenants {
        cfg.Tenants[i].AdminAPISecret, _ = cm.Decrypt(cfg.Tenants[i].AdminAPISecret)
    }
    for i := range cfg.Applications {
        cfg.Applications[i].ClientSecret, _ = cm.Decrypt(cfg.Applications[i].ClientSecret)
    }

    return cfg, nil
}

// SaveConfig with automatic encryption
func (c *Config) SaveConfig() error {
    // Encrypt secrets before saving
    for i := range c.Tenants {
        c.Tenants[i].AdminAPISecret, _ = c.cryptoManager.Encrypt(c.Tenants[i].AdminAPISecret)
    }
    for i := range c.Applications {
        c.Applications[i].ClientSecret, _ = c.cryptoManager.Encrypt(c.Applications[i].ClientSecret)
    }

    // Marshal to YAML and save
    // ...
}
```

### Phase 3: Add CLI Tool
```bash
# Encrypt existing config
./uet encrypt-config

# Decrypt for manual editing
./uet decrypt-config

# Rotate encryption key
./uet rotate-key
```

### Phase 4: Update Documentation
- Update README with encryption info
- Add security best practices
- Document key management

---

## 🔧 Advanced Features (Future)

### Key Rotation
```go
// Rotate to new key
func (c *Config) RotateEncryptionKey(newCM *crypto.CryptoManager) error {
    // Decrypt with old key
    // Re-encrypt with new key
    // Save config
}
```

### Audit Logging
```go
// Log who accessed/modified secrets
func (c *Config) AuditLog(operation, user string) {
    log.Printf("[AUDIT] %s: %s accessed config", user, operation)
}
```

### Secret Expiration
```yaml
tenants:
  - admin_api_secret: "ENC[AES256_GCM,nonce,ciphertext]"
    secret_expires_at: "2024-12-31T23:59:59Z"  # ⚠️ Rotation needed
```

---

## ⚠️ Security Considerations

### Do's ✅
- ✅ Use environment variables for master key in production
- ✅ Set file permissions: `chmod 600 .uet_key config.yaml`
- ✅ Add `.uet_key` to `.gitignore`
- ✅ Rotate keys periodically
- ✅ Use external key management in production (Vault, AWS KMS)

### Don'ts ❌
- ❌ Don't commit `.uet_key` to git
- ❌ Don't use weak passwords (< 20 characters)
- ❌ Don't share the same key across environments
- ❌ Don't disable encryption after enabling it
- ❌ Don't store master key in config file

### Threat Model

**Protected Against:**
- ✅ Accidental exposure (git commits, logs)
- ✅ Filesystem access (without key)
- ✅ Backup theft (without key)
- ✅ Insider threats (need key access)

**NOT Protected Against:**
- ❌ Memory dumps while running (secrets in RAM)
- ❌ Root/admin access (can read .uet_key)
- ❌ Supply chain attacks (compromised binary)
- ❌ Social engineering (tricking key access)

**For Maximum Security:**
Use Hardware Security Module (HSM) or Cloud KMS for key storage.

---

## 📚 References

- [AES-GCM NIST Standard](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf)
- [PBKDF2 RFC 8018](https://datatracker.ietf.org/doc/html/rfc8018)
- [Go Crypto Package](https://pkg.go.dev/crypto)
- [OWASP Key Management](https://cheatsheetseries.owasp.org/cheatsheets/Key_Management_Cheat_Sheet.html)
