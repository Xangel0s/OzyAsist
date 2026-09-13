import { useEffect } from 'react';
import { api } from '../services/api';
import { useTaskStore } from '../store/taskStore';

export function useGlobalEmergencyHandler() {
  const clearTask = useTaskStore((s) => s.clearTask);

  useEffect(() => {
    let isSubscribed = true;

    const setupListener = async () => {
      try {
        const { listen } = await import('@tauri-apps/api/event');
        const unlisten = await listen('ozy:panic_kill', async () => {
          console.warn('[KILL-SWITCH] Señal de pánico recibida. Cancelando procesos...');
          try {
            await api.post('/api/tasks/emergency-kill', {});
          } catch (err) {
            console.error('Error enviando señal de pánico al backend:', err);
          } finally {
            clearTask();
          }
        });

        if (!isSubscribed) {
          unlisten();
        }
      } catch {
        // En entorno Web o sin Tauri disponible
      }
    };

    setupListener();

    return () => {
      isSubscribed = false;
    };
  }, [clearTask]);
}
