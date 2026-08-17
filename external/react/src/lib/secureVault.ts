type Bytes = Uint8Array<ArrayBuffer>;

export interface PrfRegistration {
  credentialId: string;
  prfEnabled: boolean;
}

// Base64 helpers

function bytesOf(input: ArrayBuffer | Uint8Array): Bytes {
  if (input instanceof Uint8Array) return new Uint8Array(input);
  return new Uint8Array(input);
}

function toBytes(src: BufferSource): Bytes {
  if (src instanceof ArrayBuffer) return new Uint8Array(src);
  const view = src as ArrayBufferView;
  return new Uint8Array(
    view.buffer as ArrayBuffer,
    view.byteOffset,
    view.byteLength,
  );
}

export function toBase64Url(bytes: ArrayBuffer | Uint8Array): string {
  const arr = bytesOf(bytes);
  let binary = "";
  for (let i = 0; i < arr.length; i++) binary += String.fromCharCode(arr[i]);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function fromBase64Url(value: string): Bytes {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

export function toBase64(bytes: ArrayBuffer | Uint8Array): string {
  const arr = bytesOf(bytes);
  let binary = "";
  for (let i = 0; i < arr.length; i++) binary += String.fromCharCode(arr[i]);
  return btoa(binary);
}

export function fromBase64(value: string): Bytes {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

export function randomBytes(length: number): Bytes {
  return crypto.getRandomValues(new Uint8Array(length));
}

export function generateSalt(length: number): Bytes {
  return randomBytes(length);
}

// Feature detection

export async function isPlatformAuthenticatorAvailable(): Promise<boolean> {
  if (
    typeof PublicKeyCredential === "undefined" ||
    typeof PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable !==
      "function"
  ) {
    return false;
  }
  try {
    return await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  } catch {
    return false;
  }
}

// WebAuthn PRF registration / assertion

export async function registerPrfCredential(): Promise<PrfRegistration> {
  const credential = (await navigator.credentials.create({
    publicKey: {
      rp: { name: "khayal", id: location.hostname },
      user: {
        id: randomBytes(16),
        name: "khayal-user",
        displayName: "khayal",
      },
      challenge: randomBytes(32),
      pubKeyCredParams: [{ alg: -7, type: "public-key" }],
      authenticatorSelection: {
        authenticatorAttachment: "platform",
        residentKey: "required",
        userVerification: "required",
      },
      extensions: { prf: {} },
    },
  })) as PublicKeyCredential | null;

  if (!credential) throw new Error("Registration cancelled");

  const prfEnabled = !!credential.getClientExtensionResults().prf?.enabled;
  return {
    credentialId: toBase64Url(credential.rawId),
    prfEnabled,
  };
}

export async function getPrfOutput(
  credentialId: string,
  salt: Bytes,
): Promise<Bytes> {
  const credential = (await navigator.credentials.get({
    publicKey: {
      challenge: randomBytes(32),
      allowCredentials: [
        { id: fromBase64Url(credentialId), type: "public-key" },
      ],
      userVerification: "required",
      extensions: { prf: { eval: { first: salt } } },
    },
  })) as PublicKeyCredential | null;

  if (!credential) throw new Error("Unlock cancelled");

  const output = credential.getClientExtensionResults().prf?.results?.first;
  if (!output) throw new Error("PRF unavailable for this credential");
  return toBytes(output);
}

// Key derivation

export async function deriveKeyFromPrf(prfOutput: Bytes): Promise<CryptoKey> {
  try {
    const baseKey = await crypto.subtle.importKey(
      "raw",
      prfOutput,
      "HKDF",
      false,
      ["deriveKey"],
    );
    return await crypto.subtle.deriveKey(
      {
        name: "HKDF",
        hash: "SHA-256",
        salt: new Uint8Array(0),
        info: new TextEncoder().encode("khayal-prf"),
      },
      baseKey,
      { name: "AES-GCM", length: 256 },
      false,
      ["encrypt", "decrypt"],
    );
  } catch {
    return crypto.subtle.importKey(
      "raw",
      prfOutput,
      { name: "AES-GCM" },
      false,
      ["encrypt", "decrypt"],
    );
  }
}

// AES-GCM encrypt / decrypt

export async function encryptWithKey(
  key: CryptoKey,
  plaintext: string,
): Promise<string> {
  const iv = randomBytes(12);
  const data = new TextEncoder().encode(plaintext);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    data,
  );
  const combined = new Uint8Array(iv.length + ciphertext.byteLength);
  combined.set(iv, 0);
  combined.set(new Uint8Array(ciphertext), iv.length);
  return toBase64(combined);
}

export async function decryptWithKey(
  key: CryptoKey,
  ciphertext: string,
): Promise<string> {
  const combined = fromBase64(ciphertext);
  const iv = combined.slice(0, 12);
  const data = combined.slice(12);
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    key,
    data,
  );
  return new TextDecoder().decode(plaintext);
}
