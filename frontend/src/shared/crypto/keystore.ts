// Private key material and MLS group state live behind this interface.
// Nothing crypto-related ever touches localStorage (readable by any JS).
//
// EncryptedIndexedDbKeyStore additionally encrypts every stored blob with a
// non-extractable AES-GCM CryptoKey kept in the same database: injected JS can
// ask the browser to decrypt, but it cannot exfiltrate the key itself, and a
// copied IndexedDB file is useless without the originating browser profile.

export interface KeyStore {
  get(name: string): Promise<Uint8Array | null>;
  put(name: string, value: Uint8Array): Promise<void>;
  remove(name: string): Promise<void>;
  /** Names of all entries whose name starts with `prefix`. */
  list(prefix: string): Promise<string[]>;
}

export class MemoryKeyStore implements KeyStore {
  private entries = new Map<string, Uint8Array>();

  async get(name: string): Promise<Uint8Array | null> {
    const v = this.entries.get(name);
    return v ? new Uint8Array(v) : null;
  }

  async put(name: string, value: Uint8Array): Promise<void> {
    this.entries.set(name, new Uint8Array(value));
  }

  async remove(name: string): Promise<void> {
    this.entries.delete(name);
  }

  async list(prefix: string): Promise<string[]> {
    return [...this.entries.keys()].filter((k) => k.startsWith(prefix)).sort();
  }
}

const DB_VERSION = 1;
const BLOBS = "blobs";
const META = "meta";
const LOCAL_KEY = "local-key";
const IV_BYTES = 12;

function requestToPromise<T>(req: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

export class EncryptedIndexedDbKeyStore implements KeyStore {
  private constructor(
    private db: IDBDatabase,
    private localKey: CryptoKey,
  ) {}

  static async open(dbName = "kisy-e2ee"): Promise<EncryptedIndexedDbKeyStore> {
    const openReq = indexedDB.open(dbName, DB_VERSION);
    openReq.onupgradeneeded = () => {
      const db = openReq.result;
      if (!db.objectStoreNames.contains(BLOBS)) db.createObjectStore(BLOBS);
      if (!db.objectStoreNames.contains(META)) db.createObjectStore(META);
    };
    const db = await requestToPromise(openReq);

    let localKey = (await requestToPromise(
      db.transaction(META, "readonly").objectStore(META).get(LOCAL_KEY),
    )) as CryptoKey | undefined;
    if (!localKey) {
      // extractable=false: the browser will encrypt/decrypt for us but never
      // hand out the raw key bytes, even to our own (or injected) JS.
      localKey = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, [
        "encrypt",
        "decrypt",
      ]);
      await requestToPromise(db.transaction(META, "readwrite").objectStore(META).put(localKey, LOCAL_KEY));
    }
    return new EncryptedIndexedDbKeyStore(db, localKey);
  }

  async get(name: string): Promise<Uint8Array | null> {
    const stored = (await requestToPromise(
      this.db.transaction(BLOBS, "readonly").objectStore(BLOBS).get(name),
    )) as ArrayBuffer | undefined;
    if (!stored) return null;
    const bytes = new Uint8Array(stored);
    const iv = bytes.slice(0, IV_BYTES);
    const ciphertext = bytes.slice(IV_BYTES);
    const plain = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, this.localKey, ciphertext);
    return new Uint8Array(plain);
  }

  async put(name: string, value: Uint8Array): Promise<void> {
    const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
    const ciphertext = new Uint8Array(
      await crypto.subtle.encrypt({ name: "AES-GCM", iv }, this.localKey, value as BufferSource),
    );
    const stored = new Uint8Array(IV_BYTES + ciphertext.length);
    stored.set(iv);
    stored.set(ciphertext, IV_BYTES);
    await requestToPromise(
      this.db.transaction(BLOBS, "readwrite").objectStore(BLOBS).put(stored.buffer, name),
    );
  }

  async remove(name: string): Promise<void> {
    await requestToPromise(this.db.transaction(BLOBS, "readwrite").objectStore(BLOBS).delete(name));
  }

  async list(prefix: string): Promise<string[]> {
    const keys = (await requestToPromise(
      this.db.transaction(BLOBS, "readonly").objectStore(BLOBS).getAllKeys(),
    )) as string[];
    return keys.filter((k) => k.startsWith(prefix)).sort();
  }
}
