export type QueuedActionKind = 'dispatch' | 'deliver';

export interface QueuedAction {
  id: string;
  orderId: string;
  action: QueuedActionKind;
  body: Record<string, unknown> | undefined;
  idempotencyKey: string;
  createdAt: number;
}

export interface ActionStore {
  all(): Promise<QueuedAction[]>;
  put(action: QueuedAction): Promise<void>;
  remove(id: string): Promise<void>;
}

export type ActionSender = (action: QueuedAction) => Promise<number>;

const DB_NAME = 'depotin';
const STORE_NAME = 'actions';
const RETRY_STATUS = 500;

export function memoryStore(): ActionStore {
  const items = new Map<string, QueuedAction>();
  return {
    all: async () => [...items.values()],
    put: async (action) => {
      items.set(action.id, action);
    },
    remove: async (id) => {
      items.delete(id);
    }
  };
}

function awaitRequest<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function openDatabase(): Promise<IDBDatabase> {
  const request = indexedDB.open(DB_NAME, 1);
  request.onupgradeneeded = () => request.result.createObjectStore(STORE_NAME, { keyPath: 'id' });
  return awaitRequest(request);
}

export function indexedDbStore(): ActionStore {
  let database: Promise<IDBDatabase> | undefined;

  async function objects(mode: IDBTransactionMode): Promise<IDBObjectStore> {
    database ??= openDatabase();
    return (await database).transaction(STORE_NAME, mode).objectStore(STORE_NAME);
  }

  return {
    all: async () => awaitRequest((await objects('readonly')).getAll()),
    put: async (action) => {
      await awaitRequest((await objects('readwrite')).put(action));
    },
    remove: async (id) => {
      await awaitRequest((await objects('readwrite')).delete(id));
    }
  };
}

export class OfflineQueue {
  private flushing: Promise<number> | null = null;

  constructor(
    private readonly store: ActionStore,
    private readonly send: ActionSender
  ) {}

  async enqueue(action: Omit<QueuedAction, 'id' | 'createdAt'>): Promise<void> {
    await this.store.put({ ...action, id: crypto.randomUUID(), createdAt: Date.now() });
  }

  async count(): Promise<number> {
    return (await this.store.all()).length;
  }

  flush(): Promise<number> {
    this.flushing ??= this.drain().finally(() => {
      this.flushing = null;
    });
    return this.flushing;
  }

  private async drain(): Promise<number> {
    const pending = (await this.store.all()).sort((a, b) => a.createdAt - b.createdAt);
    for (const action of pending) {
      let status: number;
      try {
        status = await this.send(action);
      } catch {
        break;
      }
      if (status >= RETRY_STATUS) break;
      await this.store.remove(action.id);
    }
    return this.count();
  }
}
