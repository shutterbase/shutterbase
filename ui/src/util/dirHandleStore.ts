// Per-config persistence of the picked download directory. Directory handles
// are structured-cloneable, so IndexedDB can store them across sessions —
// localStorage cannot. Permissions do not survive a browser restart; the
// caller re-requests them via ensurePermission (needs a user gesture).
const DB_NAME = "shutterbase-download";
const STORE = "dirHandles";

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => request.result.createObjectStore(STORE);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function tx<T>(mode: IDBTransactionMode, run: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return openDb().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const request = run(db.transaction(STORE, mode).objectStore(STORE));
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
      }),
  );
}

export async function getStoredDirHandle(configId: string): Promise<FileSystemDirectoryHandle | undefined> {
  try {
    return await tx("readonly", (store) => store.get(configId) as IDBRequest<FileSystemDirectoryHandle | undefined>);
  } catch {
    return undefined;
  }
}

export async function storeDirHandle(configId: string, handle: FileSystemDirectoryHandle): Promise<void> {
  try {
    await tx("readwrite", (store) => store.put(handle, configId));
  } catch {
    // best effort — worst case the user re-picks the folder next run
  }
}

export async function removeDirHandle(configId: string): Promise<void> {
  try {
    await tx("readwrite", (store) => store.delete(configId));
  } catch {
    // ignore
  }
}

// ensurePermission re-acquires readwrite access on a stored handle. Returns
// false when the user denies or the handle is stale (folder deleted/moved).
export async function ensurePermission(handle: FileSystemDirectoryHandle): Promise<boolean> {
  const h = handle as any;
  try {
    if ((await h.queryPermission({ mode: "readwrite" })) === "granted") return true;
    return (await h.requestPermission({ mode: "readwrite" })) === "granted";
  } catch {
    return false;
  }
}
