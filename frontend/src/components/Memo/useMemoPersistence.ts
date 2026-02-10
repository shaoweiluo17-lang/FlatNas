import { openDB, type IDBPDatabase } from 'idb';
import { ref, type Ref } from 'vue';

const DB_NAME = 'flatnas-memo-db';
const STORE_NAME = 'memos';
const DB_VERSION = 1;

interface MemoData {
  id: string | number;
  content: string; // The rich text HTML
  simple?: string; // Plain text
  mode: 'simple' | 'rich';
  updatedAt: number;
  checksum: string;
}

// Simple checksum (DJB2)
function generateChecksum(str: string): string {
  let hash = 5381;
  for (let i = 0; i < str.length; i++) {
    hash = (hash * 33) ^ str.charCodeAt(i);
  }
  return (hash >>> 0).toString(16);
}

// Mock Sentry
const reportError = (error: unknown, context: string) => {
  console.error(`[Sentry Report] ${context}:`, error);
  const sentry = (window as {
    Sentry?: {
      captureException: (err: unknown, options?: { tags?: Record<string, string> }) => void;
    };
  }).Sentry;
  if (sentry) sentry.captureException(error, { tags: { context } });
};

let dbPromise: Promise<IDBPDatabase> | null = null;

function getDB() {
  if (!dbPromise) {
    dbPromise = openDB(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME, { keyPath: 'id' });
        }
      },
    });
  }
  return dbPromise;
}

export function useMemoPersistence(
  widgetId: string | number,
  localData: Ref<string>,
  mode: Ref<'simple' | 'rich'>
) {
  const status = ref<'idle' | 'saving' | 'success' | 'error'>('idle');
  const progress = ref(0);

  const saveToIndexedDB = async (retryCount = 0) => {
    status.value = 'saving';
    progress.value = 30;

    try {
      const db = await getDB();
      const content = localData.value;
      const checksum = generateChecksum(content);
      
      const data: MemoData = {
        id: widgetId,
        content,
        mode: mode.value,
        updatedAt: Date.now(),
        checksum
      };

      progress.value = 60;
      await db.put(STORE_NAME, data);
      
      // Verify
      const saved = await db.get(STORE_NAME, widgetId);
      if (!saved || saved.checksum !== checksum) {
        throw new Error('Checksum validation failed');
      }

      progress.value = 100;
      status.value = 'success';
      
      // Reset status after animation
      setTimeout(() => {
        status.value = 'idle';
        progress.value = 0;
      }, 1200);

    } catch (e) {
      console.error(`Save failed (attempt ${retryCount + 1})`, e);
      if (retryCount < 3) {
        setTimeout(() => saveToIndexedDB(retryCount + 1), 500 * (retryCount + 1));
      } else {
        status.value = 'error';
        reportError(e, 'MemoPersistenceSave');
      }
    }
  };

  const loadFromIndexedDB = async () => {
    try {
      const db = await getDB();
      const data = await db.get(STORE_NAME, widgetId);
      if (data) {
        // Validate checksum
        const currentChecksum = generateChecksum(data.content);
        if (currentChecksum === data.checksum) {
          localData.value = data.content;
          mode.value = data.mode;
        } else {
            reportError(new Error('Data corruption detected on load'), 'MemoPersistenceLoad');
        }
      }
    } catch (e) {
      reportError(e, 'MemoPersistenceLoad');
    }
  };

  return {
    saveToIndexedDB,
    loadFromIndexedDB,
    status,
    progress
  };
}
