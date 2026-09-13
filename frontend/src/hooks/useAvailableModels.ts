import { useState, useEffect, useCallback } from "react";
import { api } from "../services/api";

export interface ModelOption {
  id: string;
  name: string;
  provider: string;
  group: string;
  is_local: boolean;
  is_default?: boolean;
}

export function useAvailableModels() {
  const [models, setModels] = useState<ModelOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchModels = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<ModelOption[]>("/models/available");
      setModels(Array.isArray(data) ? data : []);
      setError(null);
    } catch (err: unknown) {
      const errorMsg = err instanceof Error ? err.message : "Error detectando modelos";
      setError(errorMsg);
      setModels([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return { models, loading, error, refetch: fetchModels };
}
